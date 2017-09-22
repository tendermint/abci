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

package abcicli

import (
	"sync"

	types "github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
)

type localClient struct {
	cmn.BaseService
	mtx *sync.Mutex
	types.Application
	Callback
}

func NewLocalClient(mtx *sync.Mutex, app types.Application) *localClient {
	if mtx == nil {
		mtx = new(sync.Mutex)
	}
	cli := &localClient{
		mtx:         mtx,
		Application: app,
	}
	cli.BaseService = *cmn.NewBaseService(nil, "localClient", cli)
	return cli
}

func (app *localClient) SetResponseCallback(cb Callback) {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	app.Callback = cb
}

// TODO: change types.Application to include Error()?
func (app *localClient) Error() error {
	return nil
}

func (app *localClient) FlushAsync() *ReqRes {
	// Do nothing
	return newLocalReqRes(types.ToRequestFlush(), nil)
}

func (app *localClient) EchoAsync(msg string) *ReqRes {
	return app.callback(
		types.ToRequestEcho(msg),
		types.ToResponseEcho(msg),
	)
}

func (app *localClient) InfoAsync() *ReqRes {
	app.mtx.Lock()
	resInfo := app.Application.Info()
	app.mtx.Unlock()
	return app.callback(
		types.ToRequestInfo(),
		types.ToResponseInfo(resInfo),
	)
}

func (app *localClient) SetOptionAsync(key string, value string) *ReqRes {
	app.mtx.Lock()
	log := app.Application.SetOption(key, value)
	app.mtx.Unlock()
	return app.callback(
		types.ToRequestSetOption(key, value),
		types.ToResponseSetOption(log),
	)
}

func (app *localClient) DeliverTxAsync(tx []byte) *ReqRes {
	app.mtx.Lock()
	res := app.Application.DeliverTx(tx)
	app.mtx.Unlock()
	return app.callback(
		types.ToRequestDeliverTx(tx),
		types.ToResponseDeliverTx(res.Code, res.Data, res.Log),
	)
}

func (app *localClient) CheckTxAsync(tx []byte) *ReqRes {
	app.mtx.Lock()
	res := app.Application.CheckTx(tx)
	app.mtx.Unlock()
	return app.callback(
		types.ToRequestCheckTx(tx),
		types.ToResponseCheckTx(res.Code, res.Data, res.Log),
	)
}

func (app *localClient) QueryAsync(reqQuery types.RequestQuery) *ReqRes {
	app.mtx.Lock()
	resQuery := app.Application.Query(reqQuery)
	app.mtx.Unlock()
	return app.callback(
		types.ToRequestQuery(reqQuery),
		types.ToResponseQuery(resQuery),
	)
}

func (app *localClient) CommitAsync() *ReqRes {
	app.mtx.Lock()
	res := app.Application.Commit()
	app.mtx.Unlock()
	return app.callback(
		types.ToRequestCommit(),
		types.ToResponseCommit(res.Code, res.Data, res.Log),
	)
}

func (app *localClient) InitChainAsync(params types.RequestInitChain) *ReqRes {
	app.mtx.Lock()
	app.Application.InitChain(params)
	reqRes := app.callback(
		types.ToRequestInitChain(params),
		types.ToResponseInitChain(),
	)
	app.mtx.Unlock()
	return reqRes
}

func (app *localClient) BeginBlockAsync(params types.RequestBeginBlock) *ReqRes {
	app.mtx.Lock()
	app.Application.BeginBlock(params)
	app.mtx.Unlock()
	return app.callback(
		types.ToRequestBeginBlock(params),
		types.ToResponseBeginBlock(),
	)
}

func (app *localClient) EndBlockAsync(height uint64) *ReqRes {
	app.mtx.Lock()
	resEndBlock := app.Application.EndBlock(height)
	app.mtx.Unlock()
	return app.callback(
		types.ToRequestEndBlock(height),
		types.ToResponseEndBlock(resEndBlock),
	)
}

//-------------------------------------------------------

func (app *localClient) FlushSync() error {
	return nil
}

func (app *localClient) EchoSync(msg string) (res types.Result) {
	return types.OK.SetData([]byte(msg))
}

func (app *localClient) InfoSync() (resInfo types.ResponseInfo, err error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	resInfo = app.Application.Info()
	return resInfo, nil
}

func (app *localClient) SetOptionSync(key string, value string) (res types.Result) {
	app.mtx.Lock()
	log := app.Application.SetOption(key, value)
	app.mtx.Unlock()
	return types.OK.SetLog(log)
}

func (app *localClient) DeliverTxSync(tx []byte) (res types.Result) {
	app.mtx.Lock()
	res = app.Application.DeliverTx(tx)
	app.mtx.Unlock()
	return res
}

func (app *localClient) CheckTxSync(tx []byte) (res types.Result) {
	app.mtx.Lock()
	res = app.Application.CheckTx(tx)
	app.mtx.Unlock()
	return res
}

func (app *localClient) QuerySync(reqQuery types.RequestQuery) (resQuery types.ResponseQuery, err error) {
	app.mtx.Lock()
	resQuery = app.Application.Query(reqQuery)
	app.mtx.Unlock()
	return resQuery, nil
}

func (app *localClient) CommitSync() (res types.Result) {
	app.mtx.Lock()
	res = app.Application.Commit()
	app.mtx.Unlock()
	return res
}

func (app *localClient) InitChainSync(params types.RequestInitChain) (err error) {
	app.mtx.Lock()
	app.Application.InitChain(params)
	app.mtx.Unlock()
	return nil
}

func (app *localClient) BeginBlockSync(params types.RequestBeginBlock) (err error) {
	app.mtx.Lock()
	app.Application.BeginBlock(params)
	app.mtx.Unlock()
	return nil
}

func (app *localClient) EndBlockSync(height uint64) (resEndBlock types.ResponseEndBlock, err error) {
	app.mtx.Lock()
	resEndBlock = app.Application.EndBlock(height)
	app.mtx.Unlock()
	return resEndBlock, nil
}

//-------------------------------------------------------

func (app *localClient) callback(req *types.Request, res *types.Response) *ReqRes {
	app.Callback(req, res)
	return newLocalReqRes(req, res)
}

func newLocalReqRes(req *types.Request, res *types.Response) *ReqRes {
	reqRes := NewReqRes(req)
	reqRes.Response = res
	reqRes.SetDone()
	return reqRes
}
