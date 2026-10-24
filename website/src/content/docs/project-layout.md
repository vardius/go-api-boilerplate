---
title: Project layout
description: Where each piece of code lives, what each folder may import, and which files carry build tags.
---

The layout follows one rule: code is grouped by what it is about, not by its technical layer. A service is one folder under `internal/`, and its binary is a thin `main.go` under `cmd/`.

```txt
cmd/
  auth/      main.go, store_*.go
  user/      main.go, store_*.go
internal/
  platform/  the shared kit
  auth/      tokens, OAuth2, JWKS
  user/      accounts, event-sourced
proto/       Connect contracts
gen/         generated Connect code
api/         OpenAPI files
web/         the web app and gateway
e2e/         end-to-end tests
deploy/      the Helm chart, compose extras
website/     this site
docs/standards/
.agents/skills/
```

## What each folder holds

| Folder | Holds |
|---|---|
| `cmd/<service>/main.go` | The composition root, kept under 150 lines. It loads config, opens the store and starts the servers. `<service> migrate` applies migrations or indexes and exits; `<service> env` prints the configuration reference; `<service> healthcheck` asks the local `/readyz`, for container health checks. |
| `cmd/<service>/store_<backend>.go` | Opens the chosen backend and returns its stores, readiness checks and migrate function. Apart from the end-to-end suite, these are the only files with build tags. |
| `internal/platform/` | The kit both services build on. Each package has one job: `es` (event sourcing), `outbox`, `uow` (transactions), `apperr` (error kinds), `memdb`, `mysql` and `mongodb` (backends), `httpx`, `authn`, `connectx`, `config`, `obs` and `run`. |
| `internal/user/` | The event-sourced service: `domain` (the aggregate and its events), `app` (use cases), `store` (its contract suite and one package per backend), `mail`, `httpapi`, and `rpc` (the Connect server auth calls). |
| `internal/auth/` | The state-stored token service: `app` (sessions, OAuth2 clients, grants and consent), `keys` (signing keys, JWKS and tokens), `store` (its contract suite and one package per backend) and `httpapi`. It calls the user service through the generated client in `gen/`. |
| `proto/` | Connect service definitions, linted and checked for breaking changes with buf. |
| `gen/` | Code generated from `proto/`. It is committed, so a build needs no generator. |
| `api/` | One OpenAPI 3.1 file per service. A test checks each against its router. |
| `web/` | The Vite app. Its nginx image is also the gateway, locally and in production. |
| `e2e/` | End-to-end tests that drive the running stack through the gateway, built with the `e2e` tag. |
| `deploy/` | The Helm chart in `helm/go-api-boilerplate`, for both services and the web app, and the files the compose files mount in `compose/`. |
| `docs/standards/` | The code bar every change meets: shared rules, Go and TypeScript. |
| `.agents/skills/` | Checklists for coding agents: add a command, an endpoint, a service. |

## Import rules

These are checked by a test in `internal/archtest`, so breaking one fails the build:

- `internal/platform` imports no service package.
- `internal/user/domain` imports only the standard library, `internal/platform/es` and `internal/platform/apperr`. The domain can be read and tested without any infrastructure.
- Services never import each other. They share only `internal/platform` and `gen/`, the generated Connect stubs.
- Test helpers, the `…test` packages and testcontainers, are never imported by production code.

## Build tags

Each binary links exactly one store. The tags are `persistence_mysql` and `persistence_mongodb`; memory is the default, built when neither is set. Tags appear only in `cmd/<service>/store_<backend>.go`, never inside a package, so every package compiles the same way whatever store you pick. CI builds and tests all three variants.

```sh
go build -tags persistence_mysql ./cmd/user
```
