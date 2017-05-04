package abciserver

import (
	"bytes"
	"fmt"

	abcicli "github.com/tendermint/abci/client"
	"github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"
	"github.com/tendermint/tmlibs/process"
)

func SetOption(client abcicli.Client, key, value string) {
	res := client.SetOptionSync(key, value)
	_, _, log := res.Code, res.Data, res.Log
	if res.IsErr() {
		panic(fmt.Sprintf("setting %v=%v: \nlog: %v", key, value, log))
	}
}

func Commit(client abcicli.Client, hashExp []byte) {
	res := client.CommitSync()
	_, data, log := res.Code, res.Data, res.Log
	if res.IsErr() {
		panic(fmt.Sprintf("committing %v\nlog: %v", log))
	}
	if !bytes.Equal(res.Data, hashExp) {
		panic(fmt.Sprintf("Commit hash was unexpected. Got %X expected %X",
			data, hashExp))
	}
}

func DeliverTx(client abcicli.Client, txBytes []byte, codeExp types.CodeType, dataExp []byte) {
	res := client.DeliverTxSync(txBytes)
	code, data, log := res.Code, res.Data, res.Log
	if code != codeExp {
		panic(fmt.Sprintf("DeliverTx response code was unexpected. Got %v expected %v. Log: %v",
			code, codeExp, log))
	}
	if !bytes.Equal(data, dataExp) {
		panic(fmt.Sprintf("DeliverTx response data was unexpected. Got %X expected %X",
			data, dataExp))
	}
}

func CheckTx(client abcicli.Client, txBytes []byte, codeExp types.CodeType, dataExp []byte) {
	res := client.CheckTxSync(txBytes)
	code, data, log := res.Code, res.Data, res.Log
	if res.IsErr() {
		panic(fmt.Sprintf("checking tx %X: %v\nlog: %v", txBytes, log))
	}
	if code != codeExp {
		panic(fmt.Sprintf("CheckTx response code was unexpected. Got %v expected %v. Log: %v",
			code, codeExp, log))
	}
	if !bytes.Equal(data, dataExp) {
		panic(fmt.Sprintf("CheckTx response data was unexpected. Got %X expected %X",
			data, dataExp))
	}
}
