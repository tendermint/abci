package tmspcli

import (
	//	"net"
	"reflect"
	"testing"
	"time"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/example/dummy"
	"github.com/tendermint/tmsp/server"
	"github.com/tendermint/tmsp/types"
)

func TestDummy(t *testing.T) {
	testStreamRestart(t, dummy.NewDummyApplication())
}

func TestGRPC(t *testing.T) {
	testGRPCSyncRestart(t, types.NewGRPCApplication(dummy.NewDummyApplication()))
}

func testStreamRestart(t *testing.T, app types.Application) {

	numAppendTxs := 2000
	logFreq := numAppendTxs // (reduce to get logs)

	// Start the listener
	server, err := server.NewSocketServer("unix://test.sock", app)
	if err != nil {
		Exit(Fmt("Error starting socket server: %v", err.Error()))
	}
	defer server.Stop()

	// Connect to the socket
	client, err := NewSocketClient("unix://test.sock", false)
	if err != nil {
		Exit(Fmt("Error starting socket client: %v", err.Error()))
	}
	connChan := make(chan struct{})
	client.SetConnectCallback(func() {
		go func() { connChan <- struct{}{} }()
	})
	client.Start()
	defer client.Stop()

	done := make(chan struct{})
	client.SetResponseCallback(createCallback(t, 0, numAppendTxs, logFreq, done))

	<-connChan

	// Write requests
	for counter := 0; counter < numAppendTxs; counter++ {
		if counter%(logFreq) == 0 {
			log.Warn(Fmt("%d", counter))
		}
		// Send request
		reqRes := client.AppendTxAsync([]byte("test"))
		_ = reqRes
		// check err ?

		// Sometimes send flush messages
		if counter%123 == 0 {
			client.FlushAsync()
			// check err ?
		}
	}

	server.Stop()
	server.Reset()
	time.Sleep(time.Second * 1)
	server.Start()
	// wait to restart
	<-connChan

	client.SetResponseCallback(createCallback(t, 0, numAppendTxs, logFreq, done))

	// Write requests
	for counter := 0; counter < numAppendTxs; counter++ {
		if counter%(logFreq) == 0 {
			log.Warn(Fmt("%d", counter))
		}
		// Send request
		reqRes := client.AppendTxAsync([]byte("test"))
		_ = reqRes
		// check err ?

		// Sometimes send flush messages
		if counter%123 == 0 {
			client.FlushAsync()
			// check err ?
		}
	}

	// Send final flush message
	client.FlushAsync()

	<-done
}

func createCallback(t *testing.T, counter, numAppendTxs, logFreq int, done chan struct{}) Callback {
	return func(req *types.Request, res *types.Response) {
		// Process response
		switch r := res.Value.(type) {
		case *types.Response_AppendTx:
			counter += 1
			if r.AppendTx.Code != types.CodeType_OK {
				t.Error("AppendTx failed with ret_code", r.AppendTx.Code)
			}
			if counter > numAppendTxs {
				t.Fatalf("Too many AppendTx responses. Got %d, expected %d", counter, numAppendTxs)
			}
			if counter%(logFreq) == 0 {
				log.Notice("received", "counter", counter)
			}
			if counter == numAppendTxs {
				go func() {
					time.Sleep(time.Second * 2) // Wait for a bit to allow counter overflow
					close(done)
				}()
				return
			}
		case *types.Response_Flush:
			// ignore
		default:
			t.Error("Unexpected response type", reflect.TypeOf(res.Value))
		}
	}
}

//-------------------------
// test grpc

func testGRPCSyncRestart(t *testing.T, app *types.GRPCApplication) {

	numAppendTxs := 20

	// Start the listener
	server, err := server.NewGRPCServer("unix://test.sock", app)
	if err != nil {
		Exit(Fmt("Error starting GRPC server: %v", err.Error()))
	}
	defer server.Stop()

	client, err := NewGRPCClient("unix://test.sock", false)
	if err != nil {
		Exit(Fmt("Error starting GRPC server: %v", err.Error()))
	}
	connChan := make(chan struct{})
	client.SetConnectCallback(func() {
		go func() { connChan <- struct{}{} }()
	})
	client.Start()
	<-connChan

	// Write requests
	for counter := 0; counter < numAppendTxs; counter++ {
		// Send request
		response := client.AppendTxSync([]byte("test"))
		// TODO: check err

		if response.Code != types.CodeType_OK {
			t.Error("AppendTx failed with ret_code", response.Code)
		}
		if counter > numAppendTxs {
			t.Fatal("Too many AppendTx responses")
		}
		t.Log("response", counter)
		if counter == numAppendTxs {
			go func() {
				time.Sleep(time.Second * 2) // Wait for a bit to allow counter overflow
			}()
		}

	}

	server.Stop()
	server.Reset()

	// so we notice its dead
	for {
		response := client.AppendTxSync([]byte("test"))
		if response.Code != 0 {
			break
		}
	}

	time.Sleep(time.Second * 1)
	server.Start()

	<-connChan

	// Write requests
	for counter := 0; counter < numAppendTxs; counter++ {
		// Send request
		response := client.AppendTxSync([]byte("test"))
		// TODO: check err

		if response.Code != types.CodeType_OK {
			t.Fatal("AppendTx failed with ret_code", response.Code)
		}
		if counter > numAppendTxs {
			t.Fatal("Too many AppendTx responses")
		}
		t.Log("response", counter)
		if counter == numAppendTxs {
			go func() {
				time.Sleep(time.Second * 2) // Wait for a bit to allow counter overflow
			}()
		}
	}

}
