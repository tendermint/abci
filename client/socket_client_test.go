package abcicli

import (
	"testing"

	abcicli "github.com/tendermint/abci/client"
)

func TestSocketComponentParsing(t *testing.T) {
	socket := "127.0.0.1:46658"

	// Create socket client
	client := abcicli.NewSocketClient(socket, true)
	err := client.OnStart()
	if err.Error() != "Missing protocol (ex. tcp://) in 127.0.0.1:46658" {
		t.Fatalf("Expected protocol parsing to fail: %s", err.Error())
	}
}
