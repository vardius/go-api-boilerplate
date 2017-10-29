FROM golang:latest AS build-env

MAINTAINER Rafał Lorenz <vardius@gmail.com>

ARG BIN
ENV BIN=${BIN}

ARG PKG
ENV PKG=${PKG}

COPY . /go/src/$PKG
WORKDIR /go/src/$PKG

RUN go get -u github.com/golang/dep/cmd/dep
RUN dep init
RUN dep ensure -update

RUN bootstart.sh

FROM scratch
WORKDIR /app
COPY --from=build-env /go/src/$PKG/cmd/$BIN/$BIN /app/$BIN
ENTRYPOINT ["./$BIN"]
