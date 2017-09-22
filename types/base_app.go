// Copyright 2017 Tendermint. All Rights Reserved.
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

package types

type BaseApplication struct {
}

func NewBaseApplication() *BaseApplication {
	return &BaseApplication{}
}

func (app *BaseApplication) Info() (resInfo ResponseInfo) {
	return
}

func (app *BaseApplication) SetOption(key string, value string) (log string) {
	return ""
}

func (app *BaseApplication) DeliverTx(tx []byte) Result {
	return NewResultOK(nil, "")
}

func (app *BaseApplication) CheckTx(tx []byte) Result {
	return NewResultOK(nil, "")
}

func (app *BaseApplication) Commit() Result {
	return NewResultOK([]byte("nil"), "")
}

func (app *BaseApplication) Query(reqQuery RequestQuery) (resQuery ResponseQuery) {
	return
}

func (app *BaseApplication) InitChain(reqInitChain RequestInitChain) {
}

func (app *BaseApplication) BeginBlock(reqBeginBlock RequestBeginBlock) {
}

func (app *BaseApplication) EndBlock(height uint64) (resEndBlock ResponseEndBlock) {
	return
}
