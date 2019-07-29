FROM golang:1.12.5 AS buildenv

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ARG VERSION
ARG GIT_COMMIT

ENV BIN=${BIN}
ENV VERSION=${VERSION}
ENV GIT_COMMIT=${GIT_COMMIT}

ENV GO111MODULE=on
ENV CGO_ENABLED=0

WORKDIR /app
ADD . /app

RUN go mod download
RUN go test ./...
RUN go mod verify

RUN go build -ldflags "-X main.version=$VERSION" -ldflags "-X main.gitCommit=$GIT_COMMIT" -a -o /go/bin/app ./cmd/"$BIN"

FROM scratch
COPY --from=buildenv /go/bin/app /go/bin/app
ENTRYPOINT ["/go/bin/app"]
