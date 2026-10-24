# syntax=docker/dockerfile:1

# One image definition for every Go service and store:
#   docker build --build-arg SERVICE=user --build-arg STORE=mysql .
# STORE picks the persistence build tag; memory is the default because no
# build constraint mentions persistence_memory.

FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie AS build
ARG SERVICE
ARG STORE=memory
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN test -n "$SERVICE" || (echo "set --build-arg SERVICE=auth or SERVICE=user" && false)
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY gen ./gen
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -tags "persistence_${STORE}" \
      -ldflags "-s -w -X main.version=${VERSION}" \
      -o /out/service ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian13:nonroot
ARG SERVICE
ARG STORE=memory
ARG VERSION=dev
LABEL org.opencontainers.image.source="https://github.com/vardius/go-api-boilerplate" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.title="go-api-boilerplate-${SERVICE}" \
      org.opencontainers.image.description="The ${SERVICE} service of go-api-boilerplate, built with the ${STORE} store" \
      org.opencontainers.image.version="${VERSION}"
COPY --from=build /out/service /service
# Numeric, so Kubernetes runAsNonRoot can verify it (65532 is distroless nonroot).
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/service"]
