package tmspcli

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/types"
)

const (
	OK  = types.CodeType_OK
	LOG = ""
)

const reqQueueSize = 256        // TODO make configurable
const maxResponseSize = 1048576 // 1MB TODO make configurable
const flushThrottleMS = 20      // Don't wait longer than...

// This is goroutine-safe, but users should beware that
// the application in general is not meant to be interfaced
// with concurrent callers.
type socketClient struct {
	QuitService
	wg *sync.WaitGroup

	reqQueue    chan *ReqRes
	flushTimer  *ThrottleTimer
	mustConnect bool

	mtx     sync.Mutex
	addr    string
	conn    net.Conn
	err     error
	reqSent *list.List
	resCb   func(*types.Request, *types.Response) // listens to all callbacks

	connectCallback func() // runs after connecting
}

// if mustConnect, client will start and not attempt reconnects.
// if !mustConnect, consumer must call Start(), client will always attempt reconnect
func NewSocketClient(addr string, mustConnect bool) (*socketClient, error) {
	cli := &socketClient{
		wg:          new(sync.WaitGroup),
		reqQueue:    make(chan *ReqRes, reqQueueSize),
		flushTimer:  NewThrottleTimer("socketClient", flushThrottleMS),
		mustConnect: mustConnect,

		addr:    addr,
		reqSent: list.New(),
		resCb:   nil,
	}
	cli.QuitService = *NewQuitService(log, "socketClient", cli)

	if mustConnect {
		_, err := cli.Start()
		return cli, err
	}
	return cli, nil
}

func (cli *socketClient) OnStart() error {
	cli.QuitService.OnStart()

	var err error
	var conn net.Conn
RETRY_LOOP:
	for {
		conn, err = Connect(cli.addr)
		if err != nil {
			if cli.mustConnect {
				return err
			} else {
				log.Warn(Fmt("tmsp.socketClient failed to connect to %v.  Retrying...", cli.addr))
				time.Sleep(time.Second * 3)
				continue RETRY_LOOP
			}
		}
		cli.mtx.Lock()
		cli.err = nil
		cli.conn = conn
		cli.mtx.Unlock()

		cli.wg.Add(2)
		go cli.sendRequestsRoutine(conn)
		go cli.recvResponseRoutine(conn)

		// signal that we're now connected
		cli.ConnectCallback()
		return nil
	}
	return nil // never happens
}

func (cli *socketClient) OnStop() {
	cli.QuitService.OnStop()

	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	if cli.conn != nil {
		cli.conn.Close()
	}
}

// Allow client to be reset
func (cli *socketClient) OnReset() error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	// NOTE: we dont reset err here because it needs to be checked
	cli.conn = nil
	cli.flushQueue()
	cli.reqSent = list.New()
	return nil
}

// Stop the client and restart it in a new go-routine
func (cli *socketClient) StopForError(err error) {
	if !cli.IsRunning() || !cli.IsConnected() {
		return
	}

	cli.setError(err)

	log.Warn(Fmt("Stopping tmsp.socketClient for error: %v", err.Error()))
	if stopped := cli.Stop(); !stopped {
		// if already stopped, don't reset
		return
	}

	if cli.mustConnect {
		return
	}

	// err is usually io.EOF but sometimes "read unix @sock: read: connection reset by peer"
	// since we don't really know all the ways it could fail, always reconnect.
	// need a go-routine so we can wait for callers of StopForError to return.
	go cli.restart()
}

func (cli *socketClient) restart() {

	// wait for the previous routines to finish shutting down
	cli.wg.Wait()

	log.Notice("Reset and Start client")
	cli.Reset()
	cli.Start()
}

func (cli *socketClient) IsConnected() bool {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	return cli.conn != nil
}

func (cli *socketClient) setError(err error) {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	if cli.err == nil {
		cli.err = err
	}
}

func (cli *socketClient) Error() error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	return cli.err
}

// Set listener for all responses
// NOTE: callback may get internally generated flush responses.
func (cli *socketClient) SetResponseCallback(resCb Callback) {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	cli.resCb = resCb
}

// Set a callback to be called when we connect
func (cli *socketClient) SetConnectCallback(f func()) {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	cli.connectCallback = f
}

// Call the connect callback
func (cli *socketClient) ConnectCallback() {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	if cli.connectCallback != nil {
		cli.connectCallback()
	}
}

//----------------------------------------

func (cli *socketClient) sendRequestsRoutine(conn net.Conn) {
	defer cli.wg.Done()

	w := bufio.NewWriter(conn)
	for {
		select {
		case <-cli.flushTimer.Ch:
			select {
			case cli.reqQueue <- NewReqRes(types.ToRequestFlush()):
			default:
				// Probably will fill the buffer, or retry later.
			}
		case <-cli.QuitService.Quit:
			return
		case reqres := <-cli.reqQueue:
			cli.willSendReq(reqres)
			err := types.WriteMessage(reqres.Request, w)
			if err != nil {
				cli.StopForError(fmt.Errorf("Error writing msg: %v", err))
				return
			}
			// log.Debug("Sent request", "requestType", reflect.TypeOf(reqres.Request), "request", reqres.Request)
			if _, ok := reqres.Request.Value.(*types.Request_Flush); ok {
				err = w.Flush()
				if err != nil {
					cli.StopForError(fmt.Errorf("Error flushing writer: %v", err))
					return
				}
			}
		}
	}
}

func (cli *socketClient) recvResponseRoutine(conn net.Conn) {
	defer cli.wg.Done()

	r := bufio.NewReader(conn) // Buffer reads
	for {
		var res = &types.Response{}
		err := types.ReadMessage(r, res)
		if err != nil {
			cli.StopForError(err)
			return
		}
		switch r := res.Value.(type) {
		case *types.Response_Exception:
			// XXX After setting cli.err, release waiters (e.g. reqres.Done())
			cli.StopForError(errors.New(r.Exception.Error))
			return
		default:
			// log.Debug("Received response", "responseType", reflect.TypeOf(res), "response", res)
			err := cli.didRecvResponse(res)
			if err != nil {
				cli.StopForError(err)
				return
			}
		}
	}
}

func (cli *socketClient) willSendReq(reqres *ReqRes) {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	cli.reqSent.PushBack(reqres)
}

func (cli *socketClient) didRecvResponse(res *types.Response) error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()

	// Get the first ReqRes
	next := cli.reqSent.Front()
	if next == nil {
		return fmt.Errorf("Unexpected result type %v when nothing expected", reflect.TypeOf(res.Value))
	}
	reqres := next.Value.(*ReqRes)
	if !resMatchesReq(reqres.Request, res) {
		return fmt.Errorf("Unexpected result type %v when response to %v expected",
			reflect.TypeOf(res.Value), reflect.TypeOf(reqres.Request.Value))
	}

	reqres.Response = res    // Set response
	reqres.Done()            // Release waiters
	cli.reqSent.Remove(next) // Pop first item from linked list

	// Notify reqRes listener if set
	if cb := reqres.GetCallback(); cb != nil {
		cb(res)
	}

	// Notify client listener if set
	if cli.resCb != nil {
		cli.resCb(reqres.Request, res)
	}

	return nil
}

//----------------------------------------

func (cli *socketClient) EchoAsync(msg string) *ReqRes {
	return cli.queueRequest(types.ToRequestEcho(msg))
}

func (cli *socketClient) FlushAsync() *ReqRes {
	return cli.queueRequest(types.ToRequestFlush())
}

func (cli *socketClient) InfoAsync() *ReqRes {
	return cli.queueRequest(types.ToRequestInfo())
}

func (cli *socketClient) SetOptionAsync(key string, value string) *ReqRes {
	return cli.queueRequest(types.ToRequestSetOption(key, value))
}

func (cli *socketClient) AppendTxAsync(tx []byte) *ReqRes {
	return cli.queueRequest(types.ToRequestAppendTx(tx))
}

func (cli *socketClient) CheckTxAsync(tx []byte) *ReqRes {
	return cli.queueRequest(types.ToRequestCheckTx(tx))
}

func (cli *socketClient) QueryAsync(query []byte) *ReqRes {
	return cli.queueRequest(types.ToRequestQuery(query))
}

func (cli *socketClient) CommitAsync() *ReqRes {
	return cli.queueRequest(types.ToRequestCommit())
}

func (cli *socketClient) InitChainAsync(validators []*types.Validator) *ReqRes {
	return cli.queueRequest(types.ToRequestInitChain(validators))
}

func (cli *socketClient) BeginBlockAsync(height uint64) *ReqRes {
	return cli.queueRequest(types.ToRequestBeginBlock(height))
}

func (cli *socketClient) EndBlockAsync(height uint64) *ReqRes {
	return cli.queueRequest(types.ToRequestEndBlock(height))
}

//----------------------------------------

func (cli *socketClient) EchoSync(msg string) (res types.Result) {
	reqres := cli.queueRequest(types.ToRequestEcho(msg))
	cli.FlushSync()
	if cli.err != nil {
		return types.ErrInternalError.SetLog(cli.err.Error())
	}
	resp := reqres.Response.GetEcho()
	return types.Result{Code: OK, Data: []byte(resp.Message), Log: LOG}
}

func (cli *socketClient) FlushSync() error {
	reqRes := cli.queueRequest(types.ToRequestFlush())
	if reqRes == nil {
		return fmt.Errorf("Remote app is not running")

	}
	reqRes.Wait() // NOTE: if we don't flush the queue, its possible to get stuck here
	return cli.err
}

func (cli *socketClient) InfoSync() (res types.Result) {
	reqres := cli.queueRequest(types.ToRequestInfo())
	cli.FlushSync()
	if cli.err != nil {
		return types.ErrInternalError.SetLog(cli.err.Error())
	}
	resp := reqres.Response.GetInfo()
	return types.Result{Code: OK, Data: []byte(resp.Info), Log: LOG}
}

func (cli *socketClient) SetOptionSync(key string, value string) (res types.Result) {
	reqres := cli.queueRequest(types.ToRequestSetOption(key, value))
	cli.FlushSync()
	if cli.err != nil {
		return types.ErrInternalError.SetLog(cli.err.Error())
	}
	resp := reqres.Response.GetSetOption()
	return types.Result{Code: OK, Data: nil, Log: resp.Log}
}

func (cli *socketClient) AppendTxSync(tx []byte) (res types.Result) {
	reqres := cli.queueRequest(types.ToRequestAppendTx(tx))
	cli.FlushSync()
	if cli.err != nil {
		return types.ErrInternalError.SetLog(cli.err.Error())
	}
	resp := reqres.Response.GetAppendTx()
	return types.Result{Code: resp.Code, Data: resp.Data, Log: resp.Log}
}

func (cli *socketClient) CheckTxSync(tx []byte) (res types.Result) {
	reqres := cli.queueRequest(types.ToRequestCheckTx(tx))
	cli.FlushSync()
	if cli.err != nil {
		return types.ErrInternalError.SetLog(cli.err.Error())
	}
	resp := reqres.Response.GetCheckTx()
	return types.Result{Code: resp.Code, Data: resp.Data, Log: resp.Log}
}

func (cli *socketClient) QuerySync(query []byte) (res types.Result) {
	reqres := cli.queueRequest(types.ToRequestQuery(query))
	cli.FlushSync()
	if cli.err != nil {
		return types.ErrInternalError.SetLog(cli.err.Error())
	}
	resp := reqres.Response.GetQuery()
	return types.Result{Code: resp.Code, Data: resp.Data, Log: resp.Log}
}

func (cli *socketClient) CommitSync() (res types.Result) {
	reqres := cli.queueRequest(types.ToRequestCommit())
	cli.FlushSync()
	if cli.err != nil {
		return types.ErrInternalError.SetLog(cli.err.Error())
	}
	resp := reqres.Response.GetCommit()
	return types.Result{Code: resp.Code, Data: resp.Data, Log: resp.Log}
}

func (cli *socketClient) InitChainSync(validators []*types.Validator) (err error) {
	cli.queueRequest(types.ToRequestInitChain(validators))
	cli.FlushSync()
	if cli.err != nil {
		return cli.err
	}
	return nil
}

func (cli *socketClient) BeginBlockSync(height uint64) (err error) {
	cli.queueRequest(types.ToRequestBeginBlock(height))
	cli.FlushSync()
	if cli.err != nil {
		return cli.err
	}
	return nil
}

func (cli *socketClient) EndBlockSync(height uint64) (validators []*types.Validator, err error) {
	reqres := cli.queueRequest(types.ToRequestEndBlock(height))
	cli.FlushSync()
	if cli.err != nil {
		return nil, cli.err
	}
	return reqres.Response.GetEndBlock().Diffs, nil
}

//----------------------------------------

// XXX: we may be queuing requests while the app is down
// that it wont know what to do with when it restarts.
// Should we check connected, or just flush on start ..
func (cli *socketClient) queueRequest(req *types.Request) *ReqRes {
	reqres := NewReqRes(req)

	// TODO: set cli.err if reqQueue times out
	cli.reqQueue <- reqres

	// Maybe auto-flush, or unset auto-flush
	switch req.Value.(type) {
	case *types.Request_Flush:
		cli.flushTimer.Unset()
	default:
		cli.flushTimer.Set()
	}

	return reqres
}

func (cli *socketClient) flushQueue() {
LOOP:
	for {
		select {
		case reqres := <-cli.reqQueue:
			reqres.Done()
		default:
			break LOOP
		}
	}
}

//----------------------------------------

func resMatchesReq(req *types.Request, res *types.Response) (ok bool) {
	switch req.Value.(type) {
	case *types.Request_Echo:
		_, ok = res.Value.(*types.Response_Echo)
	case *types.Request_Flush:
		_, ok = res.Value.(*types.Response_Flush)
	case *types.Request_Info:
		_, ok = res.Value.(*types.Response_Info)
	case *types.Request_SetOption:
		_, ok = res.Value.(*types.Response_SetOption)
	case *types.Request_AppendTx:
		_, ok = res.Value.(*types.Response_AppendTx)
	case *types.Request_CheckTx:
		_, ok = res.Value.(*types.Response_CheckTx)
	case *types.Request_Commit:
		_, ok = res.Value.(*types.Response_Commit)
	case *types.Request_Query:
		_, ok = res.Value.(*types.Response_Query)
	case *types.Request_InitChain:
		_, ok = res.Value.(*types.Response_InitChain)
	case *types.Request_EndBlock:
		_, ok = res.Value.(*types.Response_EndBlock)
	}
	return ok
}
