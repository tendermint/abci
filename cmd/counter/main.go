package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"

	"github.com/tendermint/abci/example/counter"
	"github.com/tendermint/abci/server"
	"github.com/tendermint/abci/types"
)

// ListenAddr is the port the abci app serves upon
const ListenAddr = "tcp://0.0.0.0:46658"

// create application, run it, print error and return code on failure
func main() {
	app := counter.NewCounterApplication(false)
	err := startApp(app)
	if err != nil {
		fmt.Printf("%#v\n", err)
		os.Exit(1)
	}
}

// launch app to listen on ListenAddr
func startApp(app types.Application) error {
	// wrap the application structure in a server that handles all network connections
	svr, err := server.NewServer(ListenAddr, "socket", app)
	if err != nil {
		return errors.Wrap(err, "Creating listener")
	}

	// create a logger that writes to stdout, you could use stderr or file
	// also, NewTMJSONLogger will output logs in json format, nice for ELK
	logger := log.NewTMLogger(os.Stdout)
	// only report Info or above, could be Debug or Error... should be set
	// via command-line flags in a real application
	logger = log.NewFilter(logger, log.AllowInfo())
	// we add info to the logger, so we can separate these log messages from other
	// messages from the same application (eg. from the db if there were one)
	logger = logger.With("module", "abci-server")
	// now, make the abci server use this logger to output messages
	svr.SetLogger(logger)

	// Start server and wait forever
	_, err = svr.Start()
	if err != nil {
		return errors.Wrap(err, "Starting server")
	}
	cmn.TrapSignal(func() {
		// Cleanup on kill server
		svr.Stop()
	})
	return nil
}
