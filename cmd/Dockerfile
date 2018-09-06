FROM golang:1.11.0 AS build-env

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ENV BIN=${BIN}

ENV GO111MODULE=on

COPY . /app
WORKDIR /app

RUN go mod download
RUN go test ./...
RUN go mod verify

RUN go build -o /go/bin/app ./cmd/"$BIN"

FROM scratch
COPY --from=build-env /go/bin/app /go/bin/app
ENTRYPOINT ["/go/bin/app"]
