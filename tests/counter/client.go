package counter

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"

	crypto "github.com/tendermint/go-crypto"

	cmn "github.com/tendermint/tmlibs/common"

	abcicli "github.com/tendermint/abci/client"
	"github.com/tendermint/abci/types"
)

// TestClient is used to test the implementation of the
type TestClient struct {
	abcicli.Client
}

// NewTestClient returns a client that can be used to test a counter application.
func NewTestClient(client abcicli.Client) *TestClient {
	return &TestClient{client}
}

// InitChain tests the implementation of InitChain on a counter application.
func (tc *TestClient) InitChain() error {
	total := 10
	vals := make([]*types.Validator, total)
	for i := 0; i < total; i++ {
		pubkey := crypto.GenPrivKeyEd25519FromSecret([]byte(cmn.Fmt("test%d", i))).PubKey().Bytes()
		power := cmn.RandInt()
		vals[i] = &types.Validator{pubkey, int64(power)}
	}
	_, err := tc.InitChainSync(types.RequestInitChain{Validators: vals})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed InitChain: %v", err))
	}
	return nil
}

// SetOption tests the implementation of SetOption on a counter application.
func (tc *TestClient) SetOption(key, value string) error {
	res, err := tc.SetOptionSync(types.RequestSetOption{Key: key, Value: value})
	log := res.GetLog()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed SetOption: setting %v=%v: log: %v", key,
			value, log))
	}
	return nil
}

// Commit tests the implementation of Commit on a counter application.
func (tc *TestClient) Commit(hashExp []byte) error {
	res, err := tc.CommitSync()
	_, data := res.Code, res.Data
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed Commit: committing %v", res.GetLog()))
	}
	if !bytes.Equal(data, hashExp) {
		return errors.Wrap(err, fmt.Sprintf("Failed Commit: got %X expected %X", data.Bytes(),
			hashExp))
	}
	return nil
}

// DeliverTx tests the implementation of DeliverTx on a counter application.
func (tc *TestClient) DeliverTx(txBytes []byte, codeExp uint32, dataExp []byte) error {
	res, err := tc.DeliverTxSync(txBytes)
	code, data, log := res.Code, res.Data, res.Log
	if code != codeExp {
		return errors.Wrap(err, fmt.Sprintf("Failed DeliverTx: got %v expected %v. log: %v", code,
			codeExp, log))
	}
	if !bytes.Equal(data, dataExp) {
		return errors.Wrap(err, fmt.Sprintf("Failed DeliverTx: got %X expected %X", data, dataExp))
	}
	return nil
}

// CheckTx tests the implementation of CheckTx on a counter application.
func (tc *TestClient) CheckTx(txBytes []byte, codeExp uint32, dataExp []byte) error {
	res, err := tc.CheckTxSync(txBytes)
	code, data, log := res.Code, res.Data, res.Log
	if code != codeExp {
		return errors.Wrap(err, fmt.Sprintf("Failed CheckTx: got %v expected %v. log: %v",
			code, codeExp, log))
	}
	if !bytes.Equal(data, dataExp) {
		return errors.Wrap(err, fmt.Sprintf("Failed CheckTx: Got %X expected %X", data, dataExp))
	}
	return nil
}
