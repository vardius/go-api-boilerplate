#!/bin/sh

go get -u github.com/golang/dep/cmd/dep
dep init
dep ensure -update

go get github.com/derekparker/delve/cmd/dlv
dlv debug ./cmd/"$BIN" -l 0.0.0.0:2345 --headless --log
