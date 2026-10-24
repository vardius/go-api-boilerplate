---
title: REST API
description: Every public route of the auth and user services, with the authentication each needs and what it returns.
---

The OpenAPI 3.1 files in `api/` are the source of truth for the REST API, and a test checks each against its service's router. This page will be generated from them. Until then, it lists the routes from the design.

<!-- verify: replace this page with the reference generated from api/auth.openapi.yaml and api/user.openapi.yaml once they exist -->

Paths are shown as the browser calls them, through the gateway. The gateway strips the prefix, so the user service sees `/api/user/v1/me` as `/v1/me`. Errors are [problem details](../../concepts/errors/), except on the OAuth endpoints.

## user

| Method | Path | Needs | Returns |
|---|---|---|---|
| `GET` | `/api/user/v1/config` | nothing | `{google_enabled, mail_preview_url?}` |
| `POST` | `/api/user/v1/login-links` | nothing; rate limited per IP and per email | `{email, next?}` → `202 {}`, always; a malformed email is `400` |
| `GET` | `/api/user/v1/me` | a user's token with `user:read` | `{id, email, google_linked, created_at}` |
| `POST` | `/api/user/v1/me/email-change` | a user's token with `user:write` | `{email}` → `202`; the code goes to the new address |
| `POST` | `/api/user/v1/me/email-change/confirm` | a user's token with `user:write` | `{code}` → `200` with the user; an email already taken is `409` |
| `GET` | `/api/user/v1/google/start` | nothing; `?link=1` needs a user | `302` to Google; `404` when Google sign-in is off |
| `GET` | `/api/user/v1/google/callback` | nothing | `302` to the web app |

## auth

| Method | Path | Needs | Returns |
|---|---|---|---|
| `GET` | `/api/auth/.well-known/jwks.json` | nothing | The current and previous public keys |
| `GET` | `/api/auth/.well-known/oauth-authorization-server` | nothing | RFC 8414 server metadata |
| `POST` | `/api/auth/v1/sessions` | nothing; rate limited | `{code}` → `201 {session: {id, expires_at}}` and the cookies; a bad code is `401` |
| `POST` | `/api/auth/v1/sessions/refresh` | the refresh cookie | `200` with rotated cookies; `401` clears them |
| `DELETE` | `/api/auth/v1/sessions/current` | the refresh cookie | `204`; revokes the grant and clears the cookies |
| `GET` | `/api/auth/v1/sessions` | a first-party session | `{sessions: [{id, created_at, last_used_at, user_agent, current}]}` |
| `DELETE` | `/api/auth/v1/sessions/{id}` | a first-party session, the owner | `204`; someone else's session is `404` |
| `GET` | `/api/auth/v1/clients` | `clients:read` | `{clients: […], limit, offset}`: the caller's own, never a secret |
| `POST` | `/api/auth/v1/clients` | `clients:write` | `{name, redirect_uris, scopes}` → `201 {client, client_secret}`; the secret is shown once |
| `DELETE` | `/api/auth/v1/clients/{id}` | `clients:write`, the owner | `204`; revokes its grants; someone else's client is `404` |
| `GET` | `/api/auth/v1/authorization-requests/{id}` | a user's session, the owner | The consent details: `{client, scopes, redirect_host}` |
| `GET` | `/api/auth/oauth/authorize` | the session cookie | Redirects; see [a third-party app](../../architecture/#a-third-party-app) |
| `POST` | `/api/auth/oauth/authorize` | the session cookie, same origin | `303` to the app's redirect URI |
| `POST` | `/api/auth/oauth/token` | client authentication | RFC 6749 token response; `invalid_client` is `401` with `WWW-Authenticate: Basic` |
| `POST` | `/api/auth/oauth/revoke` | client authentication | RFC 7009; `200` always |

## Probes

Each service answers `GET /livez` and `GET /readyz` on its own listener. See [Observability](../../concepts/observability/#health-probes).
