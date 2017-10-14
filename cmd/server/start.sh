#!/bin/sh

go get -u github.com/golang/dep/cmd/dep
dep init
dep ensure -update

if [ "$API_ENV" = 'development' ] ; then
  go get github.com/derekparker/delve/cmd/dlv
  sh -c dep ensure && dlv debug -l 0.0.0.0:2345 --headless --log
else
  go build -o app . && ./app
fi
