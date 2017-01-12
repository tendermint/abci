# Tendermint Socket Protocol (TMSP)

[![CircleCI](https://circleci.com/gh/tendermint/tmsp.svg?style=svg)](https://circleci.com/gh/tendermint/tmsp)

Blockchains are a system for creating shared multi-master application state.
**TMSP** is a socket protocol enabling a blockchain consensus engine, running in one process,
to manage a blockchain application state, running in another.

For more information on TMSP, motivations, and tutorials, please visit [our blog post](http://tendermint.com/blog/tmsp-the-tendermint-socket-protocol/).

Other implementations:
* [cpp-tmsp](https://github.com/mdyring/cpp-tmsp) by Martin Dyring-Andersen
* [js-tmsp](https://github.com/tendermint/js-tmsp)
* [jTMSP](https://github.com/jTMSP/) for Java

## Contents

This repository holds a number of important pieces:

- `types/types.proto`
	- the protobuf file defining TMSP message types, and the optional grpc interface.
        - to build, run `make protoc`
	- see `protoc --help` and [the grpc docs](https://www.grpc.io/docs) for examples and details of other languages

- golang implementation of TMSP client and server
	- two implementations:
		- asynchronous, ordered message passing over unix or tcp;
			- messages are serialized using protobuf and length prefixed
		- grpc
	- TendermintCore runs a client, and the application runs a server

- `cmd/tmsp-cli`
	- command line tool wrapping the client for probing/testing a TMSP application
	- use `tmsp-cli --version` to get the TMSP version

- examples:
	- the `cmd/counter` application, which illustrates nonce checking in txs
	- the `cmd/dummy` application, which illustrates a simple key-value merkle tree


## Message format

Since this is a streaming protocol, all messages are encoded with a length-prefix followed by the message encoded in Protobuf3.  Protobuf3 doesn't have an official length-prefix standard, so we use our own.  The first byte represents the length of the big-endian encoded length.

For example, if the Protobuf3 encoded TMSP message is `0xDEADBEEF` (4 bytes), the length-prefixed message is `0x0104DEADBEEF`.  If the Protobuf3 encoded TMSP message is 65535 bytes long, the length-prefixed message would be like `0x02FFFF...`.

Note this prefixing does not apply for grpc.

## Message types

TMSP requests/responses are simple Protobuf messages.  Check out the [schema file](https://github.com/tendermint/tmsp/blob/master/types/types.proto).

#### AppendTx
  * __Arguments__:
    * `Data ([]byte)`: The request transaction bytes
  * __Returns__:
    * `Code (uint32)`: Response code
    * `Data ([]byte)`: Result bytes, if any
    * `Log (string)`: Debug or error message
  * __Usage__:<br/>
    Append and run a transaction.  If the transaction is valid, returns CodeType.OK

#### CheckTx
  * __Arguments__:
    * `Data ([]byte)`: The request transaction bytes
  * __Returns__:
    * `Code (uint32)`: Response code
    * `Data ([]byte)`: Result bytes, if any
    * `Log (string)`: Debug or error message
  * __Usage__:<br/>
    Validate a mempool transaction, prior to broadcasting or proposing.  This message should not mutate the main state, but application
    developers may want to keep a separate CheckTx state that gets reset upon Commit.

    CheckTx can happen interspersed with AppendTx, but they happen on different connections - CheckTx from the mempool connection, and AppendTx from the consensus connection.  During Commit, the mempool is locked, so you can reset the mempool state to the latest state after running all those appendtxs, and then the mempool will re run whatever txs it has against that latest mempool stte

    Transactions are first run through CheckTx before broadcast to peers in the mempool layer.
    You can make CheckTx semi-stateful and clear the state upon `Commit` or `BeginBlock`,
    to allow for dependent sequences of transactions in the same block.

#### Commit
  * __Returns__:
    * `Data ([]byte)`: The Merkle root hash
    * `Log (string)`: Debug or error message
  * __Usage__:<br/>
    Return a Merkle root hash of the application state.

#### Query
  * __Arguments__:
    * `Data ([]byte)`: The query request bytes
  * __Returns__:
    * `Code (uint32)`: Response code
    * `Data ([]byte)`: The query response bytes
    * `Log (string)`: Debug or error message

#### Proof
  * __Arguments__:
    * `Key ([]byte)`: The key whose data you want to verifiably query
    * `Height (int64)`: The block height for which you want the proof (default=0 returns the proof for last committed block)
  * __Returns__:
    * `Code (uint32)`: Response code
    * `Data ([]byte)`: The query response bytes
    * `Log (string)`: Debug or error message
  * __Usage__:<br/>
    Return a Merkle proof from the key/value pair back to the application hash.<br/>
    *Please note* The current implementation of go-merkle doesn't support querying proofs from past blocks, so for the present moment, any height other than 0 will return an error.  Hopefully this will be improved soon(ish)

#### Flush
  * __Usage__:<br/>
    Flush the response queue.  Applications that implement `types.Application` need not implement this message -- it's handled by the project.

#### Info
  * __Returns__:
    * `Data ([]byte)`: The info bytes
  * __Usage__:<br/>
    Return information about the application state.  Application specific.

#### SetOption
  * __Arguments__:
    * `Key (string)`: Key to set
    * `Value (string)`: Value to set for key
  * __Returns__:
    * `Log (string)`: Debug or error message
  * __Usage__:<br/>
    Set application options.  E.g. Key="mode", Value="mempool" for a mempool connection, or Key="mode", Value="consensus" for a consensus connection.
    Other options are application specific.

#### InitChain
  * __Arguments__:
    * `Validators ([]Validator)`: Initial genesis validators
  * __Usage__:<br/>
    Called once upon genesis

#### BeginBlock
  * __Arguments__:
    * `Height (uint64)`: The block height that is starting
  * __Usage__:<br/>
    Signals the beginning of a new block. Called prior to any AppendTxs.

#### EndBlock
  * __Arguments__:
    * `Height (uint64)`: The block height that ended
  * __Returns__:
    * `Validators ([]Validator)`: Changed validators with new voting powers (0 to remove)
  * __Usage__:<br/>
    Signals the end of a block.  Called prior to each Commit after all transactions

## Changelog

##### Mar 26h, 2016
* Introduce BeginBlock

##### Mar 6th, 2016

* Added InitChain, EndBlock

##### Feb 14th, 2016

* s/GetHash/Commit/g
* Document Protobuf request/response fields

##### Jan 23th, 2016

* Added CheckTx/Query TMSP message types
* Added Result/Log fields to AppendTx/CheckTx/SetOption
* Removed Listener messages
* Removed Code from ResponseSetOption and ResponseGetHash
* Made examples BigEndian

##### Jan 12th, 2016

* Added "RetCodeBadNonce = 0x06" return code

##### Jan 8th, 2016

* Tendermint/TMSP now comes to consensus on the order first before AppendTx.
