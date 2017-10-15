FROM golang:latest AS build-env

MAINTAINER Rafał Lorenz <vardius@gmail.com>

ARG BIN
ENV BIN=${BIN}

COPY . /go/src/app
WORKDIR /go/src/app

RUN go-wrapper download
RUN go-wrapper install
RUN bootstart.sh

FROM scratch
WORKDIR /app
COPY --from=build-env /go/src/cmd/$BIN/$BIN /app/$BIN
ENTRYPOINT ["./$BIN"]
