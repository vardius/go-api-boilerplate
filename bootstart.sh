#!/bin/sh

if [ "$ENV" = 'development' ] ; then
  go get github.com/derekparker/delve/cmd/dlv
  dlv debug ./cmd/"$BIN" -l 0.0.0.0:2345 --headless --log
else
  go build -o ./cmd/"$BIN"/"$BIN" ./cmd/"$BIN"
fi
