FROM golang:1.17 AS buildenv

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ARG VERSION
ARG GIT_COMMIT

ENV BIN=${BIN}
ENV VERSION=${VERSION}
ENV GIT_COMMIT=${GIT_COMMIT}

ENV GO111MODULE=on
ENV CGO_ENABLED=0

# Create a location in the container for the source code.
RUN mkdir -p /app

# Copy the module files first and then download the dependencies. If this
# doesn't change, we won't need to do this again in future builds.
COPY go.* /app/

WORKDIR /app
RUN go mod download
RUN go mod verify

# Copy the source code into the container.
COPY pkg pkg
COPY cmd/"$BIN" cmd/"$BIN"

RUN go build \
    -mod=readonly \
    -tags=persistence_mysql \
    -ldflags "-X github.com/vardius/go-api-boilerplate/pkg/buildinfo.Version=$VERSION -X github.com/vardius/go-api-boilerplate/pkg/buildinfo.GitCommit=$GIT_COMMIT -X 'github.com/vardius/go-api-boilerplate/pkg/buildinfo.BuildTime=$(date -u '+%Y-%m-%d %H:%M:%S')'" \
    -a -o /go/bin/app ./cmd/"$BIN"

FROM scratch
COPY --from=buildenv /go/bin/app /go/bin/app
COPY --from=buildenv /etc/ssl/certs /etc/ssl/certs
ENTRYPOINT ["/go/bin/app"]
