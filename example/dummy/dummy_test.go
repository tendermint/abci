package dummy

import (
	"bytes"
	"io/ioutil"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	merkle "github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
	tmspcli "github.com/tendermint/tmsp/client"
	"github.com/tendermint/tmsp/server"
	"github.com/tendermint/tmsp/types"
)

func testDummy(t *testing.T, app types.Application, tx []byte, key, value string) {
	ar := app.AppendTx(tx)
	require.False(t, ar.IsErr(), ar)
	// repeating tx doesn't raise error
	ar = app.AppendTx(tx)
	require.False(t, ar.IsErr(), ar)

	// make sure query is fine
	r := app.Query([]byte(key))
	require.False(t, r.IsErr(), r)
	q := new(QueryResult)
	err := wire.ReadJSONBytes(r.Data, q)
	require.Nil(t, err)
	require.Equal(t, value, q.Value)

	// make sure proof is fine
	rp := app.Proof([]byte(key), 0)
	require.False(t, rp.IsErr(), rp)
	p, err := merkle.LoadProof(rp.Data)
	require.Nil(t, err)
	require.True(t, p.Valid())
	assert.Equal(t, []byte(key), p.Key())
	assert.Equal(t, []byte(value), p.Value())
}

func TestDummyKV(t *testing.T) {
	dummy := NewDummyApplication()
	key := "abc"
	value := key
	tx := []byte(key)
	testDummy(t, dummy, tx, key, value)

	value = "def"
	tx = []byte(key + "=" + value)
	testDummy(t, dummy, tx, key, value)
}

func TestPersistentDummyKV(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "tmsp-dummy-test") // TODO
	if err != nil {
		t.Fatal(err)
	}
	dummy := NewPersistentDummyApplication(dir)
	key := "abc"
	value := key
	tx := []byte(key)
	testDummy(t, dummy, tx, key, value)

	value = "def"
	tx = []byte(key + "=" + value)
	testDummy(t, dummy, tx, key, value)
}

func TestPersistentDummyInfo(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "tmsp-dummy-test") // TODO
	if err != nil {
		t.Fatal(err)
	}
	dummy := NewPersistentDummyApplication(dir)
	height := uint64(0)

	_, _, lastBlockInfo, _ := dummy.Info()
	if lastBlockInfo.BlockHeight != height {
		t.Fatalf("expected height of %d, got %d", height, lastBlockInfo.BlockHeight)
	}

	// make and apply block
	height = uint64(1)
	hash := []byte("foo")
	header := &types.Header{
		Height: uint64(height),
	}
	dummy.BeginBlock(hash, header)
	dummy.EndBlock(height)
	dummy.Commit()

	_, _, lastBlockInfo, _ = dummy.Info()
	if lastBlockInfo.BlockHeight != height {
		t.Fatalf("expected height of %d, got %d", height, lastBlockInfo.BlockHeight)
	}

}

// add a validator, remove a validator, update a validator
func TestValSetChanges(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "tmsp-dummy-test") // TODO
	if err != nil {
		t.Fatal(err)
	}
	dummy := NewPersistentDummyApplication(dir)

	// init with some validators
	total := 10
	nInit := 5
	vals := make([]*types.Validator, total)
	for i := 0; i < total; i++ {
		pubkey := crypto.GenPrivKeyEd25519FromSecret([]byte(Fmt("test%d", i))).PubKey().Bytes()
		power := RandInt()
		vals[i] = &types.Validator{pubkey, uint64(power)}
	}
	// iniitalize with the first nInit
	dummy.InitChain(vals[:nInit])

	vals1, vals2 := vals[:nInit], dummy.Validators()
	valsEqual(t, vals1, vals2)

	var v1, v2, v3 *types.Validator

	// add some validators
	v1, v2 = vals[nInit], vals[nInit+1]
	diff := []*types.Validator{v1, v2}
	tx1 := MakeValSetChangeTx(v1.PubKey, v1.Power)
	tx2 := MakeValSetChangeTx(v2.PubKey, v2.Power)

	makeApplyBlock(t, dummy, 1, diff, tx1, tx2)

	vals1, vals2 = vals[:nInit+2], dummy.Validators()
	valsEqual(t, vals1, vals2)

	// remove some validators
	v1, v2, v3 = vals[nInit-2], vals[nInit-1], vals[nInit]
	v1.Power = 0
	v2.Power = 0
	v3.Power = 0
	diff = []*types.Validator{v1, v2, v3}
	tx1 = MakeValSetChangeTx(v1.PubKey, v1.Power)
	tx2 = MakeValSetChangeTx(v2.PubKey, v2.Power)
	tx3 := MakeValSetChangeTx(v3.PubKey, v3.Power)

	makeApplyBlock(t, dummy, 2, diff, tx1, tx2, tx3)

	vals1 = append(vals[:nInit-2], vals[nInit+1])
	vals2 = dummy.Validators()
	valsEqual(t, vals1, vals2)

	// update some validators
	v1 = vals[0]
	if v1.Power == 5 {
		v1.Power = 6
	} else {
		v1.Power = 5
	}
	diff = []*types.Validator{v1}
	tx1 = MakeValSetChangeTx(v1.PubKey, v1.Power)

	makeApplyBlock(t, dummy, 3, diff, tx1)

	vals1 = append([]*types.Validator{v1}, vals1[1:len(vals1)]...)
	vals2 = dummy.Validators()
	valsEqual(t, vals1, vals2)

}

func makeApplyBlock(t *testing.T, dummy types.Application, heightInt int, diff []*types.Validator, txs ...[]byte) {
	// make and apply block
	height := uint64(heightInt)
	hash := []byte("foo")
	header := &types.Header{
		Height: height,
	}

	dummyChain := dummy.(types.BlockchainAware) // hmm...
	dummyChain.BeginBlock(hash, header)
	for _, tx := range txs {
		if r := dummy.AppendTx(tx); r.IsErr() {
			t.Fatal(r)
		}
	}
	diff2 := dummyChain.EndBlock(height)
	dummy.Commit()

	valsEqual(t, diff, diff2)

}

// order doesn't matter
func valsEqual(t *testing.T, vals1, vals2 []*types.Validator) {
	if len(vals1) != len(vals2) {
		t.Fatalf("vals dont match in len. got %d, expected %d", len(vals2), len(vals1))
	}
	sort.Sort(types.Validators(vals1))
	sort.Sort(types.Validators(vals2))
	for i, v1 := range vals1 {
		v2 := vals2[i]
		if !bytes.Equal(v1.PubKey, v2.PubKey) ||
			v1.Power != v2.Power {
			t.Fatalf("vals dont match at index %d. got %X/%d , expected %X/%d", i, v2.PubKey, v2.Power, v1.PubKey, v1.Power)
		}
	}
}

func makeSocketClientServer(app types.Application, name string) (tmspcli.Client, Service, error) {
	// Start the listener
	socket := Fmt("unix://%s.sock", name)
	server, err := server.NewSocketServer(socket, app)
	if err != nil {
		return nil, nil, err
	}

	// Connect to the socket
	client, err := tmspcli.NewSocketClient(socket, false)
	if err != nil {
		server.Stop()
		return nil, nil, err
	}
	client.Start()

	return client, server, err
}

func makeGRPCClientServer(app types.Application, name string) (tmspcli.Client, Service, error) {
	// Start the listener
	socket := Fmt("unix://%s.sock", name)

	gapp := types.NewGRPCApplication(app)
	server, err := server.NewGRPCServer(socket, gapp)
	if err != nil {
		return nil, nil, err
	}

	client, err := tmspcli.NewGRPCClient(socket, true)
	if err != nil {
		server.Stop()
		return nil, nil, err
	}
	return client, server, err
}

func TestClientServer(t *testing.T) {
	// set up socket app
	dummy := NewDummyApplication()
	client, server, err := makeSocketClientServer(dummy, "dummy-socket")
	require.Nil(t, err)
	defer server.Stop()
	defer client.Stop()

	runClientTests(t, client)

	// set up grpc app
	dummy = NewDummyApplication()
	gclient, gserver, err := makeGRPCClientServer(dummy, "dummy-grpc")
	require.Nil(t, err)
	defer gserver.Stop()
	defer gclient.Stop()

	runClientTests(t, gclient)
}

func runClientTests(t *testing.T, client tmspcli.Client) {
	// run some tests....
	key := "abc"
	value := key
	tx := []byte(key)
	testClient(t, client, tx, key, value)

	value = "def"
	tx = []byte(key + "=" + value)
	testClient(t, client, tx, key, value)
}

func testClient(t *testing.T, app tmspcli.Client, tx []byte, key, value string) {
	ar := app.AppendTxSync(tx)
	require.False(t, ar.IsErr(), ar)
	// repeating tx doesn't raise error
	ar = app.AppendTxSync(tx)
	require.False(t, ar.IsErr(), ar)

	// make sure query is fine
	r := app.QuerySync([]byte(key))
	require.False(t, r.IsErr(), r)
	q := new(QueryResult)
	err := wire.ReadJSONBytes(r.Data, q)
	require.Nil(t, err)
	require.Equal(t, value, q.Value)

	// make sure proof is fine
	rp := app.ProofSync([]byte(key), 0)
	require.False(t, rp.IsErr(), rp)
	p, err := merkle.LoadProof(rp.Data)
	require.Nil(t, err)
	require.True(t, p.Valid())
	assert.Equal(t, []byte(key), p.Key())
	assert.Equal(t, []byte(value), p.Value())
}
