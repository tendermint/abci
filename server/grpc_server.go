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
	"net"
	"strings"

	"google.golang.org/grpc"

	"github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
)

type GRPCServer struct {
	cmn.BaseService

	proto    string
	addr     string
	listener net.Listener
	server   *grpc.Server

	app types.ABCIApplicationServer
}

// NewGRPCServer returns a new gRPC ABCI server
func NewGRPCServer(protoAddr string, app types.ABCIApplicationServer) cmn.Service {
	parts := strings.SplitN(protoAddr, "://", 2)
	proto, addr := parts[0], parts[1]
	s := &GRPCServer{
		proto:    proto,
		addr:     addr,
		listener: nil,
		app:      app,
	}
	s.BaseService = *cmn.NewBaseService(nil, "ABCIServer", s)
	return s
}

// OnStart starts the gRPC service
func (s *GRPCServer) OnStart() error {
	s.BaseService.OnStart()
	ln, err := net.Listen(s.proto, s.addr)
	if err != nil {
		return err
	}
	s.listener = ln
	s.server = grpc.NewServer()
	types.RegisterABCIApplicationServer(s.server, s.app)
	go s.server.Serve(s.listener)
	return nil
}

// OnStop stops the gRPC server
func (s *GRPCServer) OnStop() {
	s.BaseService.OnStop()
	s.server.Stop()
}
