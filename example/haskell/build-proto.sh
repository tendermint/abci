#! /bin/bash

cd $GOPATH/src/github.com/tendermint/tmsp

# install with cabal

hprotoc -d example/haskell types/types.proto
