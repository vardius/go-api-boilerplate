FROM golang:1.11.0 AS buildenv

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ENV BIN=${BIN}

ENV GO111MODULE=on
ENV CGO_ENABLED=0

WORKDIR /app
ADD . /app

RUN go mod download
RUN go test ./...
RUN go mod verify

RUN go build -a -o /go/bin/app ./cmd/"$BIN"

RUN go get github.com/grpc-ecosystem/grpc-health-probe

FROM scratch
COPY --from=buildenv /go/bin/app /go/bin/app
COPY --from=buildenv /go/bin/grpc-health-probe /go/bin/grpc_health_probe
ENTRYPOINT ["/go/bin/app"]
