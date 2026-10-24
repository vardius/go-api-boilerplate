---
title: Changelog
description: What changed in each release of the template, newest first.
---

## Unreleased: the rewrite

A new layout on Go 1.27, built on the `next` branch and merged in one step.

<!-- verify: give this entry its version and date when next merges to master, and link the release -->

- **Two services in one module.** auth is a state-stored token service; user is event-sourced. Each has a thin `cmd/<service>/main.go` and lives in `internal/<service>/`.
- **Your own token service.** Passwordless sign-in with single-use codes, HttpOnly cookies, ES256 access tokens served through JWKS, refresh-token rotation with reuse detection, and OAuth2 with PKCE S256, client credentials and a consent screen.
- **Explicit REST routes** described in OpenAPI 3.1, with RFC 9457 problem details. The generic dispatch endpoints, the public user list and Facebook login are gone.
- **A small event-sourcing kit** in `internal/platform/es`: versioned event names, upcasts, an optimistic version check, and projections in the append transaction.
- **A transactional outbox** for email, delivered at least once with backoff.
- **Memory, MySQL and MongoDB** behind one contract suite each, picked by a build tag in `cmd/` only.
- **Local development with Docker Compose** and Mailpit, with no cluster or hosts-file edits. Kubernetes through one Helm chart.
- **Observability** with `log/slog` and OpenTelemetry.
- **AGENTS.md files and written standards** for working with coding agents.
- **This website**, which replaces the wiki.

Moving a fork or a client across: [Migrating from the 2022 layout](../migrating-from-2022/).

## October 2026: security fixes for the 2022 layout

Fixes to authorization, token handling and a crash on cancelled requests in the 2022 code, with a regression test for each, and CI on GitHub Actions. If you forked the 2022 layout, apply them or move to the rewrite. That code is tagged [`legacy-2022`](https://github.com/vardius/go-api-boilerplate/tree/legacy-2022).

<!-- verify: link the GitHub security advisory once it is published (plan decision 15) -->

## Earlier

Releases before 2026 are on [GitHub](https://github.com/vardius/go-api-boilerplate/releases).
