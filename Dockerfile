FROM golang:latest AS build-env

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ENV BIN=${BIN}

ARG PKG
ENV PKG=${PKG}

COPY . /go/src/$PKG
WORKDIR /go/src/$PKG

RUN go get -u github.com/golang/dep/cmd/dep
RUN dep init
RUN dep ensure -update

RUN go build -o ./cmd/"$BIN"/"$BIN" ./cmd/"$BIN"

FROM scratch
WORKDIR /app
COPY --from=build-env /go/src/$PKG/cmd/$BIN/$BIN /app/$BIN
ENTRYPOINT ["./$BIN"]