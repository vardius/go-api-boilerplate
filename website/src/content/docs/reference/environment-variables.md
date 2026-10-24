---
title: Environment variables
description: Every environment variable the auth and user services read, with its default and what it does.
---

Both services are configured only through the environment. [Configuration](../../guides/configuration/) explains how names are built, how secrets are handled, and how to set up keys, stores, proxies and email.

The tables below are generated from the services' config structs, by the same code behind `auth env` and `user env`. Change a description in the struct's `desc` tag, not here: everything after the marker is replaced on the next `make gen`.

<!-- verify: until the generator from plan sections C1 and F4 exists, the tables below are copied by hand from the design spec and may lag the code. Replace them with the generated output and check that `make gen` rewrites only what follows the marker -->

<!-- generated -->

## Both services

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `production` | `development` lets auth start without a signing key by generating a throwaway one. |
| `HTTP_ADDR` | `:8080` | Address of the public HTTP listener. |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | Time allowed to read request headers. |
| `HTTP_READ_TIMEOUT` | `15s` | Time allowed to read a whole request. |
| `HTTP_WRITE_TIMEOUT` | `15s` | Time allowed to write a response. |
| `HTTP_IDLE_TIMEOUT` | `60s` | How long an idle keep-alive connection stays open. |
| `HTTP_MAX_HEADER_BYTES` | `65536` | Largest request header accepted. |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error`. |
| `LOG_FORMAT` | `json` | `json` or `text`. |
| `SHUTDOWN_DELAY` | `5s` | Wait after readiness turns `503`, before servers stop. |
| `SHUTDOWN_TIMEOUT` | `20s` | Time allowed for servers and workers to stop. |
| `WEB_URL` | required | Public address of the web app, such as `http://localhost:8080`. |
| `TRUSTED_PROXIES` | empty | Comma-separated CIDRs of proxies whose `X-Forwarded-For` is believed. |
| `RATE_LIMIT_RATE` | `10` | Requests per second per client. |
| `RATE_LIMIT_BURST` | `20` | Burst size per client. |
| `RATE_LIMIT_VISITORS` | `10000` | Clients the limiter tracks at once. |
| `MIGRATE_ON_START` | | `true` applies MySQL migrations at start. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | unset | OTLP/HTTP endpoint for traces and metrics. Unset, telemetry is off. |

## auth

| Variable | Default | Description |
|---|---|---|
| `ISSUER` | required | Public URL of the auth API, such as `http://localhost:8080/api/auth`. The `iss` of every token. |
| `AUDIENCE` | `go-api-boilerplate` | The `aud` of every token. |
| `SIGNING_KEY_FILE` | | ES256 private key, PKCS#8 PEM. Required unless `APP_ENV=development`. |
| `PREVIOUS_KEYS_DIR` | | Folder of public keys still served in the JWKS, for rotation. |
| `COOKIE_SECURE` | `true` | Marks cookies `Secure`. Compose sets `false` for plain HTTP on `localhost`. |
| `REFRESH_COOKIE_PATH` | `/api/auth/v1/sessions` | Path the refresh cookie is sent to. |
| `USER_RPC_URL` | required | The user service's Connect listener, such as `http://user:8081`. |

## user

| Variable | Default | Description |
|---|---|---|
| `RPC_ADDR` | `:8081` | Address of the Connect listener. Not exposed through the gateway. |
| `AUTH_JWKS_URL` | required | Where to fetch auth's public keys. |
| `AUTH_ISSUER` | required | The `iss` a token must carry. |
| `AUTH_AUDIENCE` | `go-api-boilerplate` | The `aud` a token must carry. |
| `SMTP_ADDR` | | SMTP server, `host:port`. |
| `SMTP_USERNAME` | | SMTP user. |
| `SMTP_PASSWORD` | | SMTP password. Secret. |
| `SMTP_FROM` | | Sender address. |
| `SMTP_ALLOW_INSECURE` | `false` | `true` allows SMTP without STARTTLS. For Mailpit on your machine only. |
| `GOOGLE_CLIENT_ID` | empty | Turns Google sign-in on. |
| `GOOGLE_CLIENT_SECRET` | | Google OAuth client secret. Secret. |
| `GOOGLE_REDIRECT_URL` | | The callback registered with Google, ending in `/api/user/v1/google/callback`. |
| `MAIL_PREVIEW_URL` | | A mailbox the web app links to in development, such as Mailpit. |
| `OUTBOX_INTERVAL` | `1s` | Wait between polls once no message is due. |
| `OUTBOX_LEASE` | `2m` | How long a claimed message stays claimed. Must outlast `OUTBOX_TIMEOUT`. |
| `OUTBOX_TIMEOUT` | `30s` | Time allowed for one delivery. |
| `OUTBOX_BATCH` | `10` | Messages claimed at once. |
| `OUTBOX_MAX_ATTEMPTS` | `8` | Tries before a message is buried. |

## MySQL builds

| Variable | Default | Description |
|---|---|---|
| `MYSQL_ADDR` | `localhost:3306` | Server address. |
| `MYSQL_USER` | required | |
| `MYSQL_PASSWORD` | required | Secret. |
| `MYSQL_DATABASE` | required | Each service uses its own database. |
| `MYSQL_MAX_OPEN_CONNS` | `20` | |
| `MYSQL_MAX_IDLE_CONNS` | `10` | |
| `MYSQL_CONN_MAX_LIFETIME` | `30m` | |

## MongoDB builds

| Variable | Default | Description |
|---|---|---|
| `MONGODB_URI` | required | Connection string, with `replicaSet`. Secret. |
| `MONGODB_DATABASE` | required | Each service uses its own database. |
