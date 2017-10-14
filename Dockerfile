FROM golang:latest AS build-env

MAINTAINER Rafał Lorenz <vardius@gmail.com>

ARG BIN

COPY pkg /go/src/app/pkg
COPY cmd/$BIN /go/src/app/cmd/$BIN
WORKDIR /go/src/app/cmd/$BIN

RUN go-wrapper download
RUN go-wrapper install
RUN go build -o $BIN

FROM scratch
WORKDIR /app
COPY --from=build-env /go/src/cmd/$BIN/$BIN /app/$BIN
ENTRYPOINT ["./$BIN"]
