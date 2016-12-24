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

// client is a global variable so it can be reused by the console
var client tmspcli.Client

//structure for data passed to print response
type PR struct {
	res       types.Result
	s         string
	printCode bool
	code      string
}

func newPR(res types.Result, s string, printCode bool) PR {
	pr := PR{
		res:       res,
		s:         s,
		printCode: printCode,
		code:      "",
	}

	if printCode {
		pr.code = res.Code.String()
	}

	return pr
}

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
	fmt.Println("Unknown command:", cmd)
	fmt.Println("Please try one of the following:")
	fmt.Println("")
	cli.DefaultAppComplete(c)
}

//Generates new Args array based off of original Args call to maintain flag persistence
func persistentArgs(removeArg string, line []byte) []string {
	//function to remove first slice of matching string from string array
	remove := func(slice []string, removeText string) []string {
		for i := len(slice) - 1; i >= 0; i-- { //search from end to start
			if slice[i] == removeText {
				return append(slice[:i], slice[i+1:]...)
			}
		}
		return slice
	}

	//generate the arguments to run from orginal os.Args
	// to maintain flag arguments
	args := os.Args
	fmt.Println(args)
	args = remove(args, removeArg)
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

		args := persistentArgs("batch", line)
		err2 := app.Run(args) //cli prints error within its func call
		if err2 != nil && c.GlobalBool("mro") {
			printJSON(err)
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

		args := persistentArgs("console", line)
		err2 := app.Run(args) //cli prints error within its func call
		if err2 != nil && c.GlobalBool("mro") {
			printJSON(err)
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
	pr := newPR(res, string(res.Data), false)
	printResponse(c, pr)
	return nil
}

// Get some info from the application
func cmdInfo(c *cli.Context) error {
	res, _, _, _ := client.InfoSync()
	pr := newPR(res, string(res.Data), false)
	printResponse(c, pr)
	return nil
}

// Set an option on the application
func cmdSetOption(c *cli.Context) error {
	args := c.Args()
	if len(args) != 2 {
		return errors.New("Command set_option takes 2 arguments (key, value)")
	}
	res := client.SetOptionSync(args[0], args[1])
	pr := newPR(res, Fmt("%s=%s", args[0], args[1]), false)
	printResponse(c, pr)
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
	pr := newPR(res, string(res.Data), true)
	printResponse(c, pr)
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
	pr := newPR(res, string(res.Data), true)
	printResponse(c, pr)
	return nil
}

// Get application Merkle root hash
func cmdCommit(c *cli.Context) error {
	res := client.CommitSync()
	pr := newPR(res, Fmt("0x%X", res.Data), false)
	printResponse(c, pr)
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
	pr := newPR(res, string(res.Data), true)
	printResponse(c, pr)
	return nil
}

//--------------------------------------------------------------------------------

func printJSON(v interface{}) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		jsonBytes, _ = json.Marshal(err)
	}
	fmt.Println(string(jsonBytes))
}

func printResponse(c *cli.Context, pr PR) {

	verbose := c.GlobalBool("verbose")
	mro := c.GlobalBool("mro")

	if mro {
		jsonBytes, err := json.Marshal(pr)
		if err != nil {
			jsonBytes, _ = json.Marshal(err)
		}
		fmt.Println(string(jsonBytes))

		return
	}

	if verbose {
		fmt.Println(">", c.Command.Name, strings.Join(c.Args(), " "))
	}

	if pr.printCode {
		fmt.Printf("-> code: %s\n", pr.code)
	}

	//if pr.res.Error != "" {
	//	fmt.Printf("-> error: %s\n", pr.res.Error)
	//}

	if pr.s != "" {
		fmt.Printf("-> data: %s\n", pr.s)
	}
	if pr.res.Log != "" {
		fmt.Printf("-> log: %s\n", pr.res.Log)
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
