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

RUN GRPC_HEALTH_PROBE_VERSION=v0.2.0 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

FROM scratch
COPY --from=buildenv /go/bin/app /go/bin/app
COPY --from=buildenv /bin/grpc_health_probe /bin/grpc_health_probe
ENTRYPOINT ["/go/bin/app"]
