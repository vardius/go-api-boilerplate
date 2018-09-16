FROM golang:1.11.0 AS buildenv

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ENV BIN=${BIN}

ENV GO111MODULE=on
ENV CGO_ENABLED=0

# Secure against running as root
RUN adduser -D -u 10000 rafal
RUN mkdir /app/ && chown rafal /app/
USER rafal

WORKDIR /app
ADD . /app

RUN go mod download
RUN go test ./...
RUN go mod verify

RUN go build -a -o /go/bin/app ./cmd/"$BIN"

FROM scratch

# Secure against running as root
RUN adduser -D -u 10000 rafal
USER rafal

COPY --from=buildenv /go/bin/app /go/bin/app

ENTRYPOINT ["/go/bin/app"]
