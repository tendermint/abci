package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/client"
	"github.com/tendermint/tmsp/types"
	"github.com/urfave/cli"
)

//structure for data passed to print response
// variables must be exposed for JSON to read
type pr struct {
	Res       types.Result
	S         string
	PrintCode bool
	Code      string
}

//trivial implementation of Error for JSON prints
type er struct {
	Error string
}

func newPr(res types.Result, s string, printCode bool) *pr {
	out := &pr{
		Res:       res,
		S:         s,
		PrintCode: printCode,
		Code:      "",
	}

	if printCode {
		out.Code = res.Code.String()
	}

	return out
}

func newEr(err error) *er {
	return &er{Error: err.Error()}
}

// client is a global variable so it can be reused by the console
var client tmspcli.Client

func main() {

	//workaround for the cli library (https://github.com/urfave/cli/issues/565)
	cli.OsExiter = func(_ int) {}

	app := cli.NewApp()
	app.Name = "tmsp-cli"
	app.Usage = "tmsp-cli [command] [args...]"
	app.Version = "0.2.1" // better error handling in console
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "address",
			Value: "tcp://127.0.0.1:46658",
			Usage: "address of application socket",
		},
		cli.StringFlag{
			Name:  "tmsp",
			Value: "socket",
			Usage: "socket or grpc",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "print the command and results as if it were a console session",
		},
		cli.BoolFlag{
			Name:  "mro",
			Usage: "use machine readable output",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "batch",
			Usage: "Run a batch of tmsp commands against an application",
			Action: func(c *cli.Context) error {
				return cmdBatch(app, c)
			},
		},
		{
			Name:  "console",
			Usage: "Start an interactive tmsp console for multiple commands",
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
			Name:  "append_tx",
			Usage: "Append a new tx to application",
			Action: func(c *cli.Context) error {
				return cmdAppendTx(c)
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
		},
	}
	app.Before = before
	err := app.Run(os.Args)
	if err != nil {
		Exit(err.Error())
	}

}

func before(c *cli.Context) error {
	if client == nil {
		var err error
		client, err = tmspcli.NewClient(c.GlobalString("address"), c.GlobalString("tmsp"), false)
		if err != nil {
			Exit(err.Error())
		}
	}
	return nil
}

// badCmd is called when we invoke with an invalid first argument (just for console for now)
func badCmd(c *cli.Context, cmd string) {
	if c.GlobalBool("mro") {
		printJSON(newEr(errors.New("Unknown command")))
		return
	}

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
		err2 := app.Run(args) //cli prints error within its func call

		if err2 != nil && c.GlobalBool("mro") {
			printJSON(newEr(err2))
		}

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
		err2 := app.Run(args) //cli prints error within its func call
		if err2 != nil && c.GlobalBool("mro") {
			printJSON(newEr(err2))
		}
	}
}

// Have the application echo a message
func cmdEcho(c *cli.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return errors.New("Command echo takes 1 argument")
	}
	res := client.EchoSync(args[0])
	p := newPr(res, string(res.Data), false)
	printResponse(c, p)
	return nil
}

// Get some info from the application
func cmdInfo(c *cli.Context) error {
	res, _, _, _ := client.InfoSync()
	p := newPr(res, string(res.Data), false)
	printResponse(c, p)
	return nil
}

// Set an option on the application
func cmdSetOption(c *cli.Context) error {
	args := c.Args()
	if len(args) != 2 {
		return errors.New("Command set_option takes 2 arguments (key, value)")
	}
	res := client.SetOptionSync(args[0], args[1])
	p := newPr(res, Fmt("%s=%s", args[0], args[1]), false)
	printResponse(c, p)
	return nil
}

// Append a new tx to application
func cmdAppendTx(c *cli.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return errors.New("Command append_tx takes 1 argument")
	}
	txBytes := stringOrHexToBytes(c.Args()[0])
	res := client.AppendTxSync(txBytes)
	p := newPr(res, string(res.Data), true)
	printResponse(c, p)
	return nil
}

// Validate a tx
func cmdCheckTx(c *cli.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return errors.New("Command check_tx takes 1 argument")
	}
	txBytes := stringOrHexToBytes(c.Args()[0])
	res := client.CheckTxSync(txBytes)
	p := newPr(res, string(res.Data), true)
	printResponse(c, p)
	return nil
}

// Get application Merkle root hash
func cmdCommit(c *cli.Context) error {
	res := client.CommitSync()
	p := newPr(res, Fmt("0x%X", res.Data), false)
	printResponse(c, p)
	return nil
}

// Query application state
func cmdQuery(c *cli.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return errors.New("Command query takes 1 argument")
	}
	queryBytes := stringOrHexToBytes(c.Args()[0])
	res := client.QuerySync(queryBytes)
	p := newPr(res, string(res.Data), true)
	printResponse(c, p)
	return nil
}

//--------------------------------------------------------------------------------

func printJSON(v interface{}) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		jsonBytes, _ = json.Marshal(newEr(err))
	}
	fmt.Println(string(jsonBytes))
}

func printResponse(c *cli.Context, p *pr) {

	verbose := c.GlobalBool("verbose")
	mro := c.GlobalBool("mro")

	if mro {
		printJSON(p)
		return
	}

	if verbose {
		fmt.Println(">", c.Command.Name, strings.Join(c.Args(), " "))
	}

	if p.PrintCode {
		fmt.Printf("-> code: %s\n", p.Code)
	}

	//if pr.res.Error != "" {
	//	fmt.Printf("-> error: %s\n", pr.res.Error)
	//}

	if p.S != "" {
		fmt.Printf("-> data: %s\n", p.S)
	}
	if p.Res.Log != "" {
		fmt.Printf("-> log: %s\n", p.Res.Log)
	}

	if verbose {
		fmt.Println("")
	}

}

// NOTE: s is interpreted as a string unless prefixed with 0x
func stringOrHexToBytes(s string) []byte {
	if len(s) > 2 && s[:2] == "0x" {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			fmt.Println("Error decoding hex argument:", err.Error())
		}
		return b
	}
	return []byte(s)
}
