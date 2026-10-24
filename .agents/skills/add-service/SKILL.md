---
name: add-service
description: Add a new Go service to the repo (binary, packages, store builds, compose, Helm, CI and docs) by copying the user service's layout. Use when a new bounded context needs its own process.
---

# Add a service

A service is its own process with its own data. It talks to the others only
over Connect, through `proto/` and `gen/`. The
[add a service guide](https://vardius.github.io/go-api-boilerplate/guides/add-a-service/)
walks through it; this is the checklist. `<svc>` is the new name.

## Steps

1. **Packages** under `internal/<svc>/`, laid out like `internal/user`:
   `domain` (if event-sourced), `app`, `store` with `storetest` and the
   `memory`, `mysql` and `mongodb` backends, `httpapi`, and `rpc` if others
   call it. Add `internal/<svc>/AGENTS.md` with the rules true only there.
2. **Boundaries** in `internal/archtest/archtest_test.go`: the new service
   imports no other service, and no service imports it.
3. **Binary** `cmd/<svc>/`, copied from `cmd/user`: `main.go` (the
   composition root, under 150 lines), `config.go`, `healthcheck.go`, and
   `store_memory.go`, `store_mysql.go`, `store_mongodb.go`, the only files
   with build tags. Keep the `migrate`, `env` and `healthcheck` subcommands.
4. **Contract**, if others call it: `proto/<svc>/v1/*.proto`, then
   `make gen`. Callers use the generated client with a service token whose
   scope only that caller holds.
5. **API document** `api/<svc>.openapi.yaml`, with the OpenAPI and
   authorization matrix tests copied from `internal/user/httpapi`.
6. **Compose**: a service in `compose.yaml` (memory build, healthcheck
   `/service healthcheck`), its store settings in `compose.mysql.yaml` and
   `compose.mongodb.yaml`, a database and user in
   `deploy/compose/mysql-init.sql`, and a gateway location in
   `web/nginx.conf` if it has public routes.
7. **Helm**: a template in `deploy/helm/go-api-boilerplate/templates/`, its
   values, the migration Job list in `migrate.yaml`, and the `ci/*-values.yaml`
   files.
8. **Release and CI**: add it to the `service` matrix in
   `.github/workflows/release.yml`, and to `scripts/env-reference.sh` so its
   variables are documented.
9. **Docs**: the root `AGENTS.md` map and scope router, the README layout,
   and the website's architecture and project layout pages.

## Verify

```bash
make check
make up && make e2e
```

Run `make up STORE=mysql` and `STORE=mongodb` too.
