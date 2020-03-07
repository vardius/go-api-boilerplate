# First stage: build the executable.
FROM golang:1.14-alpine AS buildenv

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

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

# Create appuser
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

# Copy the module files first and then download the dependencies. If this
# doesn't change, we won't need to do this again in future builds.
COPY go.* /app/

WORKDIR /app
RUN go mod download
RUN go mod verify

# Copy the source code into the container.
COPY internal internal
COPY cmd/"$BIN" cmd/"$BIN"
# Copy required proto files
COPY cmd/auth/proto cmd/auth/proto

RUN go build \
    -mod=readonly \
    -ldflags "-X github.com/vardius/go-api-boilerplate/internal/buildinfo.Version=$VERSION -X github.com/vardius/go-api-boilerplate/internal/buildinfo.GitCommit=$GIT_COMMIT -X 'github.com/vardius/go-api-boilerplate/internal/buildinfo.BuildTime=$(date -u '+%Y-%m-%d %H:%M:%S')'" \
    -a -o /go/bin/app ./cmd/"$BIN"

FROM scratch

# Import from builder.
COPY --from=buildenv /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=buildenv /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=buildenv /etc/passwd /etc/passwd
COPY --from=buildenv /etc/group /etc/group

COPY --from=buildenv /go/bin/app /go/bin/app

# Use an unprivileged user.
USER appuser:appuser

ENTRYPOINT ["/go/bin/app"]
