package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	abcicli "github.com/tendermint/abci/client"
	"github.com/tendermint/abci/example/counter"
	"github.com/tendermint/abci/example/dummy"
	"github.com/tendermint/abci/server"
	"github.com/tendermint/abci/types"
	"github.com/tendermint/abci/version"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
	"github.com/urfave/cli"
)

// Structure for data passed to print response.
type response struct {
	// generic abci response
	Data []byte
	Code types.CodeType
	Log  string

	Query *queryResponse
}

type queryResponse struct {
	Key    []byte
	Value  []byte
	Height uint64
	Proof  []byte
}

// client is a global variable so it can be reused by the console
var client abcicli.Client

var logger log.Logger

func main() {

	//workaround for the cli library (https://github.com/urfave/cli/issues/565)
	cli.OsExiter = func(_ int) {}

	app := cli.NewApp()
	app.Name = "abci-cli"
	app.Usage = "abci-cli [command] [args...]"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "address",
			Value: "tcp://127.0.0.1:46658",
			Usage: "address of application socket",
		},
		cli.StringFlag{
			Name:  "abci",
			Value: "socket",
			Usage: "socket or grpc",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "print the command and results as if it were a console session",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "batch",
			Usage: "Run a batch of abci commands against an application",
			Action: func(c *cli.Context) error {
				return cmdBatch(app, c)
			},
		},
		{
			Name:  "console",
			Usage: "Start an interactive abci console for multiple commands",
			Action: func(c *cli.Context) error {
				return cmdConsole(app, c)
			},
		},
		{
			Name:  "echo",
			Usage: "Have the application echo a message",
			Action: func(c *cli.Context) error {
				return cmdEcho(c)
			},
		},
		{
			Name:  "info",
			Usage: "Get some info about the application",
			Action: func(c *cli.Context) error {
				return cmdInfo(c)
			},
		},
		{
			Name:  "set_option",
			Usage: "Set an option on the application",
			Action: func(c *cli.Context) error {
				return cmdSetOption(c)
			},
		},
		{
			Name:  "deliver_tx",
			Usage: "Deliver a new tx to application",
			Action: func(c *cli.Context) error {
				return cmdDeliverTx(c)
			},
		},
		{
			Name:  "check_tx",
			Usage: "Validate a tx",
			Action: func(c *cli.Context) error {
				return cmdCheckTx(c)
			},
		},
		{
			Name:  "commit",
			Usage: "Commit the application state and return the Merkle root hash",
			Action: func(c *cli.Context) error {
				return cmdCommit(c)
			},
		},
		{
			Name:  "query",
			Usage: "Query application state",
			Action: func(c *cli.Context) error {
				return cmdQuery(c)
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path",
					Value: "/store",
					Usage: "Path to prefix the query with",
				},
				cli.IntFlag{
					Name:  "height",
					Value: 0,
					Usage: "Height to query the blockchain at",
				},
				cli.BoolFlag{
					Name:  "prove",
					Usage: "Whether or not to return a merkle proof of the query result",
				},
			},
		},
		// app examples
		{
			Name:  "counter",
			Usage: "",
			Action: func(c *cli.Context) error {
				return cmdCounter(c)
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "serial",
					// False by default, see https://github.com/urfave/cli/issues/650
					// Value: false,
					Usage: "Enforce incrementing (serial) transactions",
				},
			},
		},
		{
			Name:  "dummy",
			Usage: "",
			Action: func(c *cli.Context) error {
				return cmdDummy(c)
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "persist",
					Value: "",
					Usage: "directory to use for a databse",
				},
			},
		},
	}
	app.Before = before
	err := app.Run(os.Args)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

}

func before(c *cli.Context) error {
	if logger == nil {
		logger = log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), log.AllowError())
	}
	if client == nil {
		var err error
		client, err = abcicli.NewClient(c.GlobalString("address"), c.GlobalString("abci"), false)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
		client.SetLogger(logger.With("module", "abci-client"))
		if _, err := client.Start(); err != nil {
			return err
		}
	}
	return nil
}

// badCmd is called when we invoke with an invalid first argument (just for console for now)
func badCmd(c *cli.Context, cmd string) {
	fmt.Println("Unknown command:", cmd)
	fmt.Println("Please try one of the following:")
	fmt.Println("")
	cli.DefaultAppComplete(c)
}

//Generates new Args array based off of previous call args to maintain flag persistence
func persistentArgs(line []byte) []string {

	//generate the arguments to run from orginal os.Args
	// to maintain flag arguments
	args := os.Args
	args = args[:len(args)-1] // remove the previous command argument

	if len(line) > 0 { //prevents introduction of extra space leading to argument parse errors
		args = append(args, strings.Split(string(line), " ")...)
	}
	return args
}

//--------------------------------------------------------------------------------

func cmdBatch(app *cli.App, c *cli.Context) error {
	bufReader := bufio.NewReader(os.Stdin)
	for {
		line, more, err := bufReader.ReadLine()
		if more {
			return errors.New("Input line is too long")
		} else if err == io.EOF {
			break
		} else if len(line) == 0 {
			continue
		} else if err != nil {
			return err
		}

		args := persistentArgs(line)
		app.Run(args) //cli prints error within its func call
	}
	return nil
}

func cmdConsole(app *cli.App, c *cli.Context) error {
	// don't hard exit on mistyped commands (eg. check vs check_tx)
	app.CommandNotFound = badCmd

	for {
		fmt.Printf("\n> ")
		bufReader := bufio.NewReader(os.Stdin)
		line, more, err := bufReader.ReadLine()
		if more {
			return errors.New("Input is too long")
		} else if err != nil {
			return err
		}

		args := persistentArgs(line)
		app.Run(args) //cli prints error within its func call
	}
}

// Have the application echo a message
func cmdEcho(c *cli.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return errors.New("Command echo takes 1 argument")
	}
	resEcho := client.EchoSync(args[0])
	printResponse(c, response{
		Data: resEcho.Data,
	})
	return nil
}

// Get some info from the application
func cmdInfo(c *cli.Context) error {
	resInfo, err := client.InfoSync()
	if err != nil {
		return err
	}
	printResponse(c, response{
		Data: []byte(resInfo.Data),
	})
	return nil
}

// Set an option on the application
func cmdSetOption(c *cli.Context) error {
	args := c.Args()
	if len(args) != 2 {
		return errors.New("Command set_option takes 2 arguments (key, value)")
	}
	resSetOption := client.SetOptionSync(args[0], args[1])
	printResponse(c, response{
		Log: resSetOption.Log,
	})
	return nil
}

// Append a new tx to application
func cmdDeliverTx(c *cli.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return errors.New("Command deliver_tx takes 1 argument")
	}
	txBytes, err := stringOrHexToBytes(c.Args()[0])
	if err != nil {
		return err
	}
	res := client.DeliverTxSync(txBytes)
	printResponse(c, response{
		Code: res.Code,
		Data: res.Data,
		Log:  res.Log,
	})
	return nil
}

// Validate a tx
func cmdCheckTx(c *cli.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return errors.New("Command check_tx takes 1 argument")
	}
	txBytes, err := stringOrHexToBytes(c.Args()[0])
	if err != nil {
		return err
	}
	res := client.CheckTxSync(txBytes)
	printResponse(c, response{
		Code: res.Code,
		Data: res.Data,
		Log:  res.Log,
	})
	return nil
}

// Get application Merkle root hash
func cmdCommit(c *cli.Context) error {
	res := client.CommitSync()
	printResponse(c, response{
		Code: res.Code,
		Data: res.Data,
		Log:  res.Log,
	})
	return nil
}

// Query application state
func cmdQuery(c *cli.Context) error {
	args := c.Args()

	if len(args) != 1 {
		return errors.New("Command query takes 1 argument, the query bytes")
	}

	queryBytes, err := stringOrHexToBytes(args[0])
	if err != nil {
		return err
	}

	path := c.String("path")
	height := c.Int("height")
	prove := c.Bool("prove")

	resQuery, err := client.QuerySync(types.RequestQuery{
		Data:   queryBytes,
		Path:   path,
		Height: uint64(height),
		Prove:  prove,
	})
	if err != nil {
		return err
	}
	printResponse(c, response{
		Code: resQuery.Code,
		Log:  resQuery.Log,
		Query: &queryResponse{
			Key:    resQuery.Key,
			Value:  resQuery.Value,
			Height: resQuery.Height,
			Proof:  resQuery.Proof,
		},
	})
	return nil
}

// Counter the example app
func cmdCounter(c *cli.Context) error {
	app := counter.NewCounterApplication(c.Bool("serial"))

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Start the listener
	srv, err := server.NewServer(c.String("address"), c.String("abci"), app)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if _, err := srv.Start(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return nil
}

// Dummy the example app
func cmdDummy(c *cli.Context) error {

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Create the application - in memory or persisted to disk
	var app types.Application
	if c.String("persist") == "" {
		app = dummy.NewDummyApplication()
	} else {
		app = dummy.NewPersistentDummyApplication(c.String("persist"))
		app.(*dummy.PersistentDummyApplication).SetLogger(logger.With("module", "dummy"))
	}

	// Start the listener
	srv, err := server.NewServer(c.String("address"), c.String("abci"), app)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if _, err := srv.Start(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})

	return nil
}

//--------------------------------------------------------------------------------

func printResponse(c *cli.Context, rsp response) {

	verbose := c.GlobalBool("verbose")

	if verbose {
		fmt.Println(">", c.Command.Name, strings.Join(c.Args(), " "))
	}

	if !rsp.Code.IsOK() {
		fmt.Printf("-> code: %s\n", rsp.Code.String())
	}
	if len(rsp.Data) != 0 {
		fmt.Printf("-> data: %s\n", rsp.Data)
		fmt.Printf("-> data.hex: %X\n", rsp.Data)
	}
	if rsp.Log != "" {
		fmt.Printf("-> log: %s\n", rsp.Log)
	}

	if rsp.Query != nil {
		fmt.Printf("-> height: %d\n", rsp.Query.Height)
		if rsp.Query.Key != nil {
			fmt.Printf("-> key: %s\n", rsp.Query.Key)
			fmt.Printf("-> key.hex: %X\n", rsp.Query.Key)
		}
		if rsp.Query.Value != nil {
			fmt.Printf("-> value: %s\n", rsp.Query.Value)
			fmt.Printf("-> value.hex: %X\n", rsp.Query.Value)
		}
		if rsp.Query.Proof != nil {
			fmt.Printf("-> proof: %X\n", rsp.Query.Proof)
		}
	}

	if verbose {
		fmt.Println("")
	}

}

// NOTE: s is interpreted as a string unless prefixed with 0x
func stringOrHexToBytes(s string) ([]byte, error) {
	if len(s) > 2 && strings.ToLower(s[:2]) == "0x" {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			err = fmt.Errorf("Error decoding hex argument: %s", err.Error())
			return nil, err
		}
		return b, nil
	}

	if !strings.HasPrefix(s, "\"") || !strings.HasSuffix(s, "\"") {
		err := fmt.Errorf("Invalid string arg: \"%s\". Must be quoted or a \"0x\"-prefixed hex string", s)
		return nil, err
	}

	return []byte(s[1 : len(s)-1]), nil
}
