---
title: Environment variables
description: Every environment variable the auth and user services read, with its default and what it does.
---

Both services are configured only through the environment. [Configuration](../../guides/configuration/) explains how names are built, how secrets are handled, and how to set up keys, stores, proxies and email.

The tables below are generated from the services' config structs, by the same code behind `auth env` and `user env`. Change a description in the struct's `desc` tag, not here: everything after the marker is replaced on the next `make gen`.

Tracing and metrics also read the standard OpenTelemetry variables, such as `OTEL_EXPORTER_OTLP_ENDPOINT`. With none set, telemetry is off.

<!-- generated -->

## auth

| Variable | Default | Required | Description |
|---|---|---|---|
| `APP_ENV` | `production` |  | Where the service runs. development lets it start without SIGNING_KEY_FILE, on a key generated at start that dies with the process. |
| `HTTP_ADDR` | `:8080` |  | Address the server listens on. |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` |  | How long a client may take to send the request headers. |
| `HTTP_READ_TIMEOUT` | `15s` |  | How long a client may take to send the whole request. |
| `HTTP_WRITE_TIMEOUT` | `15s` |  | How long writing the response may take, from the end of the request headers. |
| `HTTP_IDLE_TIMEOUT` | `60s` |  | How long a keep-alive connection may wait for its next request. |
| `HTTP_MAX_HEADER_BYTES` | `65536` |  | Largest request header block accepted, in bytes. |
| `LOG_LEVEL` | `info` |  | Lowest level logged: debug, info, warn or error. |
| `LOG_FORMAT` | `json` |  | Log format: json or text. |
| `SHUTDOWN_DELAY` | `5s` |  | How long readiness reports 503 before servers stop, so load balancers stop routing here first. |
| `SHUTDOWN_TIMEOUT` | `20s` |  | How long servers get to finish in-flight work once they stop. |
| `ISSUER` |  | yes | Public URL of the auth API, the iss of every token, such as http://localhost:8080/api/auth. |
| `AUDIENCE` | `go-api-boilerplate` |  | The aud of every access token; every service checks it. |
| `WEB_URL` |  | yes | Public URL of the web app, such as http://localhost:8080, which hosts the sign-in, consent and OAuth error pages. |
| `SIGNING_KEY_FILE` |  |  | Path of the ES256 (P-256) private key tokens are signed with, as a PKCS#8 PEM. Required unless APP_ENV is development, which then signs with a key generated at start. |
| `PREVIOUS_KEYS_DIR` |  |  | Directory of public keys (*.pem, PKIX) the service signed with before, still published in the JWKS so their tokens verify until they expire. |
| `COOKIE_SECURE` | `true` |  | Mark the session cookies Secure. Turn off only to serve plain http on localhost. |
| `REFRESH_COOKIE_PATH` | `/api/auth/v1/sessions` |  | Public path of the sessions routes: the only path the refresh cookie is sent to. |
| `USER_RPC_URL` |  | yes | Base URL of the user service's Connect API, such as http://user:8081, where sign-in codes are redeemed. |
| `TRUSTED_PROXIES` |  |  | Comma-separated CIDRs of the proxies whose X-Forwarded-For names the client, for rate limits. |
| `RATE_LIMIT_RATE` | `10` |  | Requests per second each client may make, on average. |
| `RATE_LIMIT_BURST` | `20` |  | Requests a client may make at once before the rate applies. |
| `RATE_LIMIT_VISITORS` | `10000` |  | Clients remembered at once; the one seen least recently is forgotten first. |
| `MIGRATE_ON_START` | `false` |  | Apply the store's migrations before serving. Leave off where a separate job migrates, as the Helm chart does. |

### auth with the MySQL store

Built with `-tags persistence_mysql`, it also reads:

| Variable | Default | Required | Description |
|---|---|---|---|
| `MYSQL_ADDR` | `localhost:3306` |  | MySQL server address, host:port. |
| `MYSQL_USER` |  | yes | MySQL user name. |
| `MYSQL_PASSWORD` |  | yes | Secret. MySQL password. |
| `MYSQL_DATABASE` |  | yes | MySQL database (schema) name. |
| `MYSQL_MAX_OPEN_CONNS` | `20` |  | Most connections open at once. |
| `MYSQL_MAX_IDLE_CONNS` | `10` |  | Most idle connections kept for reuse. |
| `MYSQL_CONN_MAX_LIFETIME` | `30m` |  | Longest a connection is reused before it is replaced. |

### auth with the MongoDB store

Built with `-tags persistence_mongodb`, it also reads:

| Variable | Default | Required | Description |
|---|---|---|---|
| `MONGODB_URI` |  | yes | Secret. MongoDB connection string. Transactions need a replica set, so name one: mongodb://localhost:27017/?replicaSet=rs0 |
| `MONGODB_DATABASE` |  | yes | Name of the MongoDB database the service keeps its data in |

## user

| Variable | Default | Required | Description |
|---|---|---|---|
| `HTTP_ADDR` | `:8080` |  | Address the server listens on. |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` |  | How long a client may take to send the request headers. |
| `HTTP_READ_TIMEOUT` | `15s` |  | How long a client may take to send the whole request. |
| `HTTP_WRITE_TIMEOUT` | `15s` |  | How long writing the response may take, from the end of the request headers. |
| `HTTP_IDLE_TIMEOUT` | `60s` |  | How long a keep-alive connection may wait for its next request. |
| `HTTP_MAX_HEADER_BYTES` | `65536` |  | Largest request header block accepted, in bytes. |
| `RPC_ADDR` | `:8081` |  | Address the Connect API listens on, for the auth service only: keep it off the gateway. |
| `LOG_LEVEL` | `info` |  | Lowest level logged: debug, info, warn or error. |
| `LOG_FORMAT` | `json` |  | Log format: json or text. |
| `SHUTDOWN_DELAY` | `5s` |  | How long readiness reports 503 before servers stop, so load balancers stop routing here first. |
| `SHUTDOWN_TIMEOUT` | `20s` |  | How long servers get to finish in-flight work once they stop. |
| `WEB_URL` |  | yes | The web app's public origin, which mailed links and redirects lead to. |
| `AUTH_JWKS_URL` |  | yes | URL of the auth service's JWKS, the keys access tokens are signed with. |
| `AUTH_ISSUER` |  | yes | The iss every access token must carry: the auth API's public URL. |
| `AUTH_AUDIENCE` | `go-api-boilerplate` |  | The aud every access token must include. |
| `SMTP_ADDR` |  | yes | SMTP server, host:port, such as smtp.example.com:587. |
| `SMTP_USERNAME` |  |  | SMTP user name; empty sends without logging in. |
| `SMTP_PASSWORD` |  |  | Secret. SMTP password. |
| `SMTP_FROM` |  | yes | Sender of every email: an address, or Name <address>. |
| `SMTP_ALLOW_INSECURE` | `false` |  | Send in plain text when the server doesn't offer STARTTLS. Only for a local mail catcher such as Mailpit. |
| `GOOGLE_CLIENT_ID` |  |  | OAuth client ID of the Google Cloud app. Google sign-in is off while it is empty. |
| `GOOGLE_CLIENT_SECRET` |  |  | Secret. That OAuth client's secret. Required with GOOGLE_CLIENT_ID. |
| `GOOGLE_REDIRECT_URL` |  |  | The redirect URI registered with Google: the public URL of /v1/google/callback. Required with GOOGLE_CLIENT_ID. |
| `MAIL_PREVIEW_URL` |  |  | A mail catcher's web UI for the sign-in page to link to, such as Mailpit in development. Empty in production. |
| `TRUSTED_PROXIES` |  |  | Comma-separated CIDRs of the proxies whose X-Forwarded-For names the client, as the login-link rate limit needs. |
| `RATE_LIMIT_RATE` | `10` |  | Requests per second each client may make, on average. |
| `RATE_LIMIT_BURST` | `20` |  | Requests a client may make at once before the rate applies. |
| `RATE_LIMIT_VISITORS` | `10000` |  | Clients remembered at once; the one seen least recently is forgotten first. |
| `OUTBOX_INTERVAL` | `1s` |  | Wait between outbox polls once no message is due. |
| `OUTBOX_LEASE` | `2m` |  | How long a claimed message stays claimed; must outlast the timeout. |
| `OUTBOX_TIMEOUT` | `30s` |  | Longest one delivery may take. |
| `OUTBOX_BATCH` | `10` |  | Messages claimed per poll. |
| `OUTBOX_WORKERS` | `10` |  | Messages of a batch delivered at once; ceil(batch/workers) × timeout must stay under the lease. |
| `OUTBOX_MAX_ATTEMPTS` | `8` |  | Deliveries tried before a message is buried. |
| `MIGRATE_ON_START` | `false` |  | Apply the migrations (MySQL) or ensure the indexes (MongoDB) before serving. |

### user with the MySQL store

Built with `-tags persistence_mysql`, it also reads:

| Variable | Default | Required | Description |
|---|---|---|---|
| `MYSQL_ADDR` | `localhost:3306` |  | MySQL server address, host:port. |
| `MYSQL_USER` |  | yes | MySQL user name. |
| `MYSQL_PASSWORD` |  | yes | Secret. MySQL password. |
| `MYSQL_DATABASE` |  | yes | MySQL database (schema) name. |
| `MYSQL_MAX_OPEN_CONNS` | `20` |  | Most connections open at once. |
| `MYSQL_MAX_IDLE_CONNS` | `10` |  | Most idle connections kept for reuse. |
| `MYSQL_CONN_MAX_LIFETIME` | `30m` |  | Longest a connection is reused before it is replaced. |

### user with the MongoDB store

Built with `-tags persistence_mongodb`, it also reads:

| Variable | Default | Required | Description |
|---|---|---|---|
| `MONGODB_URI` |  | yes | Secret. MongoDB connection string. Transactions need a replica set, so name one: mongodb://localhost:27017/?replicaSet=rs0 |
| `MONGODB_DATABASE` |  | yes | Name of the MongoDB database the service keeps its data in |
