# AGENTS.md

A Go API boilerplate: an event-sourced user service and a state-stored
OAuth 2 auth service, a React web app, one origin through a gateway. The
code bar is [`docs/standards/`](docs/standards/shared.md); this file holds
only what is specific to this repo.

## Map

| Path | What lives there |
| --- | --- |
| `cmd/auth`, `cmd/user` | composition roots: config, stores, servers; `serve`, `migrate`, `env`, `healthcheck` |
| `internal/platform` | the shared kit: config, logs and traces, HTTP, tokens, event sourcing, outbox, databases |
| `internal/user` | the user service: domain, use cases, HTTP and Connect, stores |
| `internal/auth` | the auth service: sessions, OAuth 2 clients and grants, signing keys, stores |
| `internal/archtest` | the import rules below, as a test |
| `proto`, `gen` | the Connect contract between the services and its generated code |
| `api` | the OpenAPI document of each service |
| `web` | the React app, served by nginx, which proxies `/api/*` |
| `e2e` | end-to-end tests against the running stack |
| `deploy` | compose files' extras and the Helm chart |
| `website` | the documentation site |

## Scope router

| Editing… | Read first |
| --- | --- |
| the shared kit | [`internal/platform/AGENTS.md`](internal/platform/AGENTS.md) |
| events, aggregates, projections | [`internal/platform/es/AGENTS.md`](internal/platform/es/AGENTS.md) |
| the user service | [`internal/user/AGENTS.md`](internal/user/AGENTS.md) |
| the auth service | [`internal/auth/AGENTS.md`](internal/auth/AGENTS.md) |
| the web app | [`web/AGENTS.md`](web/AGENTS.md) |
| Docker, compose, Helm | [`deploy/AGENTS.md`](deploy/AGENTS.md) |
| the documentation site | [`website/AGENTS.md`](website/AGENTS.md) |

Recipes for common changes are in [`.agents/skills/`](.agents/skills/):
add a command, add an endpoint, add a service.

## Project rules

- Comments only for a *why* that no name or test can hold.
- Docs live in `website/`. A change people would notice (an endpoint, a
  config variable, a flow) updates its page in the same change. Code
  samples there are imported from real files.
- Shapes that change together, in one change: `proto` → `gen` (`make gen`);
  a handler → its `api/*.openapi.yaml` → the web app's zod schema → the
  website reference; a config field → `make gen` (the configuration
  reference is generated).
- Services never import each other (`internal/archtest` enforces it). The
  auth service calls the user service only through `gen/` over Connect.
- Events are immutable once released: a change is a new `.vN` type with an
  upcaster. The user service owns its event stream and projections; the
  auth service owns its tables. Neither reads the other's database.
- Trust boundaries: the gateway is the only public entry; services verify
  every token against the auth service's JWKS; the Connect listener is
  internal and needs a service token. Loosening one (a new public route
  without auth, a wider scope, a cookie attribute) needs a test showing who
  can and can't reach it.
- Secrets never appear in logs, events, errors or the repo; config fields
  holding one are `config.Secret`.

## Workflow config

- Base branch: master
- Verify: `make check` (CI also runs the container tests and `make e2e`
  for each store)
- Run: `docker compose up --wait`, then http://localhost:8080; mail at
  http://localhost:8025
- Review rules: this file and the scoped `AGENTS.md` of every touched
  path; a violation in new code blocks.
- Merge: ask
- Tracker: GitHub issues
- Plans: the issue
