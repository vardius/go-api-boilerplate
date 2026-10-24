---
title: REST API
description: How the auth and user services' REST APIs are reached, authenticated and documented, with links to the reference generated from their OpenAPI documents.
---

The OpenAPI 3.1 documents in `api/` are the source of truth for the REST API, and each service's tests hold its router to its document: every route is documented, every documented route exists, and every response the tests see matches its schema. The reference pages are generated from those documents on every build:

- [User API](user/): sign-in links, the signed-in user, email change and Google sign-in. Source: [`api/user.openapi.yaml`](https://github.com/vardius/go-api-boilerplate/blob/master/api/user.openapi.yaml).
- [Auth API](auth/): sessions, OAuth apps, the OAuth 2 endpoints and the keys that verify every access token. Source: [`api/auth.openapi.yaml`](https://github.com/vardius/go-api-boilerplate/blob/master/api/auth.openapi.yaml).

## Paths

Every route is served through the gateway, under `/api/user` or `/api/auth`, which are the documents' servers. The gateway strips the prefix, so the user service sees `/api/user/v1/me` as `/v1/me`. The reference shows the paths as the services see them.

## Authentication

| Scheme | In | Used by |
|---|---|---|
| `bearer` | `Authorization: Bearer <access token>` | Third-party apps and other services, on both APIs |
| `cookie` | the `access_token` cookie | The web app, on both APIs |
| `refreshCookie` | the `refresh_token` cookie | The auth API's session refresh and sign-out |
| `clientBasic` | `Authorization: Basic` with the client ID and secret | OAuth apps at `/oauth/token` and `/oauth/revoke` |

Each operation lists the schemes it accepts and the scopes it needs. [Authentication](../../concepts/authentication/) explains the tokens, the scopes and the grants.

## Errors

Errors are [problem details](../../concepts/errors/), the `Problem` schema in both documents. The exceptions are `/oauth/token` and `/oauth/revoke`, which answer with the RFC 6749 `OAuthError` shape, `{"error", "error_description"}`, because that is what OAuth clients parse.

## Probes

Each service answers `GET /livez` and `GET /readyz` on its own listener. They are documented with the rest, and described in [Observability](../../concepts/observability/#health-probes).
