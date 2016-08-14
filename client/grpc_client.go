package tmspcli

import (
	"errors"
	"net"
	"sync"
	"time"

	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/types"
)

// A stripped copy of the remoteClient that makes
// synchronous calls using grpc
type grpcClient struct {
	QuitService
	mustConnect bool

	client types.TMSPApplicationClient

	mtx   sync.Mutex
	addr  string
	err   error
	resCb func(*types.Request, *types.Response) // listens to all callbacks

	connectCallback func() // runs after connecting
}

// if mustConnect, client will start and not attempt reconnects.
// if !mustConnect, consumer must call Start(), client will always attempt reconnect
func NewGRPCClient(addr string, mustConnect bool) (*grpcClient, error) {
	cli := &grpcClient{
		addr:        addr,
		mustConnect: mustConnect,
	}
	cli.QuitService = *NewQuitService(nil, "grpcClient", cli)
	if mustConnect {
		_, err := cli.Start() // Just start it, it's confusing for callers to remember to start.
		return cli, err
	}
	return cli, nil
}

func dialerFunc(addr string, timeout time.Duration) (net.Conn, error) {
	return Connect(addr)
}

func (cli *grpcClient) OnStart() error {
	cli.QuitService.OnStart()
RETRY_LOOP:

	for {
		conn, err := grpc.Dial(cli.addr, grpc.WithInsecure(), grpc.WithDialer(dialerFunc))
		if err != nil {
			if cli.mustConnect {
				return err
			} else {
				log.Warn(Fmt("tmsp.grpcClient failed to connect to %v.  Retrying...\n", cli.addr))
				time.Sleep(time.Second * 3)
				continue RETRY_LOOP
			}
		}

		client := types.NewTMSPApplicationClient(conn)

		time.Sleep(time.Second * 3)
	ENSURE_CONNECTED:
		for {
			_, err := client.Echo(context.Background(), &types.RequestEcho{"hello"}, grpc.FailFast(true))
			if err == nil {
				break ENSURE_CONNECTED
			}
			time.Sleep(time.Second)
		}

		cli.mtx.Lock()
		cli.client = client
		cli.err = nil
		cli.mtx.Unlock()

		// signal that we're now connected
		cli.ConnectCallback()
		return nil
	}
}

func (cli *grpcClient) OnStop() {
	cli.QuitService.OnStop()
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	// TODO: how to close conn? its not a net.Conn and grpc doesn't expose a Close()
	/*if cli.conn != nil {
		cli.conn.Close()
	}*/
}

func (cli *grpcClient) OnReset() error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	cli.client = nil
	return nil
}

func (cli *grpcClient) StopForError(err error) {

	if !cli.IsRunning() || !cli.IsConnected() {
		return
	}

	cli.setError(err)

	log.Warn(Fmt("Stopping tmsp.grpcClient for error: %v", err.Error()))
	if stopped := cli.Stop(); !stopped {
		// if already stopped, don't reset
		return
	}

	if cli.mustConnect {
		return
	}

	log.Notice("Reset and Start client")
	cli.Reset()
	go cli.Start()
}

func (cli *grpcClient) IsConnected() bool {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	return cli.client != nil
}

func (cli *grpcClient) setError(err error) {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	if cli.err == nil {
		cli.err = err
	}
}

func (cli *grpcClient) Error() error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	return cli.err
}

// Set listener for all responses
// NOTE: callback may get internally generated flush responses.
func (cli *grpcClient) SetResponseCallback(resCb Callback) {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	cli.resCb = resCb
}

// Called on connecting
func (cli *grpcClient) SetConnectCallback(f func()) {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	cli.connectCallback = f
}

func (cli *grpcClient) ConnectCallback() {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	if cli.connectCallback != nil {
		cli.connectCallback()
	}
}

//----------------------------------------
// GRPC calls are synchronous, but some callbacks expect to be called asynchronously
// (eg. the mempool expects to be able to lock to remove bad txs from cache).
// To accomodate, we finish each call in its own go-routine,
// which is expensive, but easy - if you want something better, use the socket protocol!
// maybe one day, if people really want it, we use grpc streams,
// but hopefully not :D

func (cli *grpcClient) EchoAsync(msg string) *ReqRes {
	req := types.ToRequestEcho(msg)
	res, err := cli.client.Echo(context.Background(), req.GetEcho(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_Echo{res}})
}

func (cli *grpcClient) FlushAsync() *ReqRes {
	req := types.ToRequestFlush()
	res, err := cli.client.Flush(context.Background(), req.GetFlush(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_Flush{res}})
}

func (cli *grpcClient) InfoAsync() *ReqRes {
	req := types.ToRequestInfo()
	res, err := cli.client.Info(context.Background(), req.GetInfo(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_Info{res}})
}

func (cli *grpcClient) SetOptionAsync(key string, value string) *ReqRes {
	req := types.ToRequestSetOption(key, value)
	res, err := cli.client.SetOption(context.Background(), req.GetSetOption(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_SetOption{res}})
}

func (cli *grpcClient) AppendTxAsync(tx []byte) *ReqRes {
	req := types.ToRequestAppendTx(tx)
	res, err := cli.client.AppendTx(context.Background(), req.GetAppendTx(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_AppendTx{res}})
}

func (cli *grpcClient) CheckTxAsync(tx []byte) *ReqRes {
	req := types.ToRequestCheckTx(tx)
	res, err := cli.client.CheckTx(context.Background(), req.GetCheckTx(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_CheckTx{res}})
}

func (cli *grpcClient) QueryAsync(query []byte) *ReqRes {
	req := types.ToRequestQuery(query)
	res, err := cli.client.Query(context.Background(), req.GetQuery(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_Query{res}})
}

func (cli *grpcClient) CommitAsync() *ReqRes {
	req := types.ToRequestCommit()
	res, err := cli.client.Commit(context.Background(), req.GetCommit(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_Commit{res}})
}

func (cli *grpcClient) InitChainAsync(validators []*types.Validator) *ReqRes {
	req := types.ToRequestInitChain(validators)
	res, err := cli.client.InitChain(context.Background(), req.GetInitChain(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_InitChain{res}})
}

func (cli *grpcClient) BeginBlockAsync(height uint64) *ReqRes {
	req := types.ToRequestBeginBlock(height)
	res, err := cli.client.BeginBlock(context.Background(), req.GetBeginBlock(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_BeginBlock{res}})
}

func (cli *grpcClient) EndBlockAsync(height uint64) *ReqRes {
	req := types.ToRequestEndBlock(height)
	res, err := cli.client.EndBlock(context.Background(), req.GetEndBlock(), grpc.FailFast(true))
	if err != nil {
		cli.err = err
	}
	return cli.finishAsyncCall(req, &types.Response{&types.Response_EndBlock{res}})
}

func (cli *grpcClient) finishAsyncCall(req *types.Request, res *types.Response) *ReqRes {
	reqres := NewReqRes(req)
	reqres.Response = res // Set response
	reqres.Done()         // Release waiters
	reqres.SetDone()      // so reqRes.SetCallback will run the callback

	// go routine for callbacks
	go func() {
		// Notify reqRes listener if set
		if cb := reqres.GetCallback(); cb != nil {
			cb(res)
		}

		// Notify client listener if set
		if cli.resCb != nil {
			cli.resCb(reqres.Request, res)
		}
	}()
	return reqres
}

func (cli *grpcClient) checkErrGetResult() *types.Result {
	if cli.err != nil {
		errorLog := cli.err.Error()
		cli.StopForError(cli.err)
		result := types.ErrInternalError
		result.SetLog(errorLog)
		return &result
	}
	return nil
}

func (cli *grpcClient) checkGetErr() error {
	if cli.err != nil {
		err := errors.New(cli.err.Error())
		cli.StopForError(cli.err)
		return err
	}
	return nil
}

//----------------------------------------

func (cli *grpcClient) EchoSync(msg string) (res types.Result) {
	reqres := cli.EchoAsync(msg)
	if res := cli.checkErrGetResult(); res != nil {
		return *res
	}
	resp := reqres.Response.GetEcho()
	return types.NewResultOK([]byte(resp.Message), LOG)
}

func (cli *grpcClient) FlushSync() error {
	return nil
}

func (cli *grpcClient) InfoSync() (res types.Result) {
	reqres := cli.InfoAsync()
	if res := cli.checkErrGetResult(); res != nil {
		return *res
	}
	resp := reqres.Response.GetInfo()
	return types.NewResultOK([]byte(resp.Info), LOG)
}

func (cli *grpcClient) SetOptionSync(key string, value string) (res types.Result) {
	reqres := cli.SetOptionAsync(key, value)
	if res := cli.checkErrGetResult(); res != nil {
		return *res
	}
	resp := reqres.Response.GetSetOption()
	return types.Result{Code: OK, Data: nil, Log: resp.Log}
}

func (cli *grpcClient) AppendTxSync(tx []byte) (res types.Result) {
	reqres := cli.AppendTxAsync(tx)
	if res := cli.checkErrGetResult(); res != nil {
		return *res
	}
	resp := reqres.Response.GetAppendTx()
	return types.Result{Code: resp.Code, Data: resp.Data, Log: resp.Log}
}

func (cli *grpcClient) CheckTxSync(tx []byte) (res types.Result) {
	reqres := cli.CheckTxAsync(tx)
	if res := cli.checkErrGetResult(); res != nil {
		return *res
	}
	resp := reqres.Response.GetCheckTx()
	return types.Result{Code: resp.Code, Data: resp.Data, Log: resp.Log}
}

func (cli *grpcClient) QuerySync(query []byte) (res types.Result) {
	reqres := cli.QueryAsync(query)
	if res := cli.checkErrGetResult(); res != nil {
		return *res
	}
	resp := reqres.Response.GetQuery()
	return types.Result{Code: resp.Code, Data: resp.Data, Log: resp.Log}
}

func (cli *grpcClient) CommitSync() (res types.Result) {
	reqres := cli.CommitAsync()
	if res := cli.checkErrGetResult(); res != nil {
		return *res
	}
	resp := reqres.Response.GetCommit()
	return types.Result{Code: resp.Code, Data: resp.Data, Log: resp.Log}
}

func (cli *grpcClient) InitChainSync(validators []*types.Validator) (err error) {
	cli.InitChainAsync(validators)
	if err := cli.checkGetErr(); err != nil {
		return err
	}
	return nil
}

func (cli *grpcClient) BeginBlockSync(height uint64) (err error) {
	cli.BeginBlockAsync(height)
	if err := cli.checkGetErr(); err != nil {
		return err
	}
	return nil
}

func (cli *grpcClient) EndBlockSync(height uint64) (validators []*types.Validator, err error) {
	reqres := cli.EndBlockAsync(height)
	if err := cli.checkGetErr(); err != nil {
		return nil, err
	}
	return reqres.Response.GetEndBlock().Diffs, nil
}
