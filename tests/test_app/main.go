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

package main

import (
	"fmt"
	"os"

	"github.com/tendermint/abci/types"
)

var abciType string

func init() {
	abciType = os.Getenv("ABCI")
	if abciType == "" {
		abciType = "socket"
	}
}

func main() {
	testCounter()
}

func testCounter() {
	abciApp := os.Getenv("ABCI_APP")
	if abciApp == "" {
		panic("No ABCI_APP specified")
	}

	fmt.Printf("Running %s test with abci=%s\n", abciApp, abciType)
	appProc := startApp(abciApp)
	defer appProc.StopProcess(true)
	client := startClient(abciType)
	defer client.Stop()

	setOption(client, "serial", "on")
	commit(client, nil)
	deliverTx(client, []byte("abc"), types.CodeType_BadNonce, nil)
	commit(client, nil)
	deliverTx(client, []byte{0x00}, types.CodeType_OK, nil)
	commit(client, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	deliverTx(client, []byte{0x00}, types.CodeType_BadNonce, nil)
	deliverTx(client, []byte{0x01}, types.CodeType_OK, nil)
	deliverTx(client, []byte{0x00, 0x02}, types.CodeType_OK, nil)
	deliverTx(client, []byte{0x00, 0x03}, types.CodeType_OK, nil)
	deliverTx(client, []byte{0x00, 0x00, 0x04}, types.CodeType_OK, nil)
	deliverTx(client, []byte{0x00, 0x00, 0x06}, types.CodeType_BadNonce, nil)
	commit(client, []byte{0, 0, 0, 0, 0, 0, 0, 5})
}
