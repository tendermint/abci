/*
Package server is used to start a new ABCI server.

It contains two server implementation:
 * gRPC server
 * socket server

*/

package server

import (
	"fmt"

	"github.com/tendermint/abci/types"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

func NewServer(protoAddr, transport string, app types.Application,
	logger log.Logger) (cmn.Service, error) {

	var s cmn.Service
	var err error
	switch transport {
	case "socket":
		s = NewSocketServer(protoAddr, app, logger)
	case "grpc":
		s = NewGRPCServer(protoAddr, types.NewGRPCApplication(app), logger)
	default:
		err = fmt.Errorf("Unknown server type %s", transport)
	}
	return s, err
}
