// Copyright 2016 Tendermint. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
)

// var maxNumberConnections = 2

type SocketServer struct {
	cmn.BaseService

	proto    string
	addr     string
	listener net.Listener

	connsMtx   sync.Mutex
	conns      map[int]net.Conn
	nextConnID int

	appMtx sync.Mutex
	app    types.Application
}

func NewSocketServer(protoAddr string, app types.Application) cmn.Service {
	parts := strings.SplitN(protoAddr, "://", 2)
	proto, addr := parts[0], parts[1]
	s := &SocketServer{
		proto:    proto,
		addr:     addr,
		listener: nil,
		app:      app,
		conns:    make(map[int]net.Conn),
	}
	s.BaseService = *cmn.NewBaseService(nil, "ABCIServer", s)
	return s
}

func (s *SocketServer) OnStart() error {
	s.BaseService.OnStart()
	ln, err := net.Listen(s.proto, s.addr)
	if err != nil {
		return err
	}
	s.listener = ln
	go s.acceptConnectionsRoutine()
	return nil
}

func (s *SocketServer) OnStop() {
	s.BaseService.OnStop()
	s.listener.Close()

	s.connsMtx.Lock()
	for id, conn := range s.conns {
		delete(s.conns, id)
		conn.Close()
	}
	s.connsMtx.Unlock()
}

func (s *SocketServer) addConn(conn net.Conn) int {
	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()

	connID := s.nextConnID
	s.nextConnID++
	s.conns[connID] = conn

	return connID
}

// deletes conn even if close errs
func (s *SocketServer) rmConn(connID int, conn net.Conn) error {
	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()

	delete(s.conns, connID)
	return conn.Close()
}

func (s *SocketServer) acceptConnectionsRoutine() {
	// semaphore := make(chan struct{}, maxNumberConnections)

	for {
		// semaphore <- struct{}{}

		// Accept a connection
		s.Logger.Info("Waiting for new connection...")
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.IsRunning() {
				return // Ignore error from listener closing.
			}
			s.Logger.Error("Failed to accept connection: " + err.Error())
		} else {
			s.Logger.Info("Accepted a new connection")
		}

		connID := s.addConn(conn)

		closeConn := make(chan error, 2)              // Push to signal connection closed
		responses := make(chan *types.Response, 1000) // A channel to buffer responses

		// Read requests from conn and deal with them
		go s.handleRequests(closeConn, conn, responses)
		// Pull responses from 'responses' and write them to conn.
		go s.handleResponses(closeConn, responses, conn)

		go func() {
			// Wait until signal to close connection
			errClose := <-closeConn
			if err == io.EOF {
				s.Logger.Error("Connection was closed by client")
			} else if errClose != nil {
				s.Logger.Error("Connection error", "error", errClose)
			} else {
				// never happens
				s.Logger.Error("Connection was closed.")
			}

			// Close the connection
			err := s.rmConn(connID, conn)
			if err != nil {
				s.Logger.Error("Error in closing connection", "error", err)
			}

			// <-semaphore
		}()
	}
}

// Read requests from conn and deal with them
func (s *SocketServer) handleRequests(closeConn chan error, conn net.Conn, responses chan<- *types.Response) {
	var count int
	var bufReader = bufio.NewReader(conn)
	for {

		var req = &types.Request{}
		err := types.ReadMessage(bufReader, req)
		if err != nil {
			if err == io.EOF {
				closeConn <- err
			} else {
				closeConn <- fmt.Errorf("Error reading message: %v", err.Error())
			}
			return
		}
		s.appMtx.Lock()
		count++
		s.handleRequest(req, responses)
		s.appMtx.Unlock()
	}
}

func (s *SocketServer) handleRequest(req *types.Request, responses chan<- *types.Response) {
	switch r := req.Value.(type) {
	case *types.Request_Echo:
		responses <- types.ToResponseEcho(r.Echo.Message)
	case *types.Request_Flush:
		responses <- types.ToResponseFlush()
	case *types.Request_Info:
		resInfo := s.app.Info()
		responses <- types.ToResponseInfo(resInfo)
	case *types.Request_SetOption:
		so := r.SetOption
		logStr := s.app.SetOption(so.Key, so.Value)
		responses <- types.ToResponseSetOption(logStr)
	case *types.Request_DeliverTx:
		res := s.app.DeliverTx(r.DeliverTx.Tx)
		responses <- types.ToResponseDeliverTx(res.Code, res.Data, res.Log)
	case *types.Request_CheckTx:
		res := s.app.CheckTx(r.CheckTx.Tx)
		responses <- types.ToResponseCheckTx(res.Code, res.Data, res.Log)
	case *types.Request_Commit:
		res := s.app.Commit()
		responses <- types.ToResponseCommit(res.Code, res.Data, res.Log)
	case *types.Request_Query:
		resQuery := s.app.Query(*r.Query)
		responses <- types.ToResponseQuery(resQuery)
	case *types.Request_InitChain:
		s.app.InitChain(*r.InitChain)
		responses <- types.ToResponseInitChain()
	case *types.Request_BeginBlock:
		s.app.BeginBlock(*r.BeginBlock)
		responses <- types.ToResponseBeginBlock()
	case *types.Request_EndBlock:
		resEndBlock := s.app.EndBlock(r.EndBlock.Height)
		responses <- types.ToResponseEndBlock(resEndBlock)
	default:
		responses <- types.ToResponseException("Unknown request")
	}
}

// Pull responses from 'responses' and write them to conn.
func (s *SocketServer) handleResponses(closeConn chan error, responses <-chan *types.Response, conn net.Conn) {
	var count int
	var bufWriter = bufio.NewWriter(conn)
	for {
		var res = <-responses
		err := types.WriteMessage(res, bufWriter)
		if err != nil {
			closeConn <- fmt.Errorf("Error writing message: %v", err.Error())
			return
		}
		if _, ok := res.Value.(*types.Response_Flush); ok {
			err = bufWriter.Flush()
			if err != nil {
				closeConn <- fmt.Errorf("Error flushing write buffer: %v", err.Error())
				return
			}
		}
		count++
	}
}
