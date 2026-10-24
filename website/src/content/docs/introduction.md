---
title: Introduction
description: What go-api-boilerplate is, why it is built the way it is, and what it deliberately leaves out.
---

go-api-boilerplate is a template for a Go API. You create your own repository from it and change what you need. It is not a framework or a library: nothing here is meant to be imported from another module.

It holds two services in one Go module, and a small web app that shows them working:

- **auth** issues tokens. It signs users in with one-time codes, runs an OAuth2 server for third-party apps, and publishes its public keys as a JWKS. It stores clients and grants as ordinary rows.
- **user** owns accounts. It is event-sourced: every change is an event in the user's stream, and a `users` table is updated in the same transaction.
- **web** is a Vite and React app, served by an nginx gateway that also forwards `/api/auth/` and `/api/user/`. The browser sees one origin.

Both services build on a kit in `internal/platform`: event sourcing, a transactional outbox, error kinds, HTTP and Connect helpers, token verification, configuration and observability. Every store has three backends, memory, MySQL and MongoDB, held to the same rules by one contract suite.

<!-- verify: web/, internal/auth, internal/user and the httpx, authn, connectx, config, obs and run packages are being written; check this summary against them before the first deploy -->

## Why it is built this way

**Event sourcing only where it pays.** The user service keeps every change as an event, because an account's history is worth having and the pattern is what people come here to copy. The auth service doesn't: OAuth clients, codes and refresh tokens are secrets with short lives, and an append-only log would keep them forever. Seeing both side by side shows when each fits.

**Tokens are checked where they arrive.** Access tokens are ES256 JWTs. Each API verifies them with auth's public keys, which it fetches once and caches, so a normal request never calls auth.

**One transaction per change.** A change's events, the read models built from them and the emails it causes commit together or not at all. The read model never lags, and no email goes out for a change that rolled back.

**Use cases are plain methods.** There is no command bus. A use case is a method you can read top to bottom, call from a test, and delete without touching a registry.

**Contracts instead of generators.** Each backend's queries are written by hand and must pass the same contract suite. The REST API is described in OpenAPI files that a test checks against the router.

## What it leaves out

These are choices, not gaps. Each has a reason, and the docs show where you would add it.

- **Asynchronous projections with checkpoints.** Projections run in the append transaction instead. When one needs to be asynchronous, move it onto the outbox: [CQRS and projections](../concepts/cqrs-and-projections/) shows how.
- **A message broker** such as NATS or Kafka. No flow between the services needs one. The outbox is the place a broker would plug in.
- **Code generators for SQL or HTTP** (sqlc, oapi-codegen, huma). About twenty queries and fifteen routes don't repay one, and the HTTP generators want to own the router.
- **A CSS framework or component kit in the web app.** Six screens on CSS modules and design tokens need neither.
- **Facebook login, generic command dispatch, a public user list.** None had a caller that needed it, and three of them were security holes.
- **Terraform.** The Helm chart is what you deploy. How you provision a cluster is your choice.

## The 2022 layout

The previous version, with its command bus, `pkg/` directory, Terraform and Telepresence setup, is tagged [`legacy-2022`](https://github.com/vardius/go-api-boilerplate/tree/legacy-2022). If you started from it, [Migrating from the 2022 layout](../reference/migrating-from-2022/) maps each endpoint and concept to its replacement.

## Next

- [Quick start](../quick-start/): run everything locally in about five minutes.
- [Architecture](../architecture/): the request flows and the trust boundaries.
- [Project layout](../project-layout/): where each piece of code lives, and the import rules between them.
