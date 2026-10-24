# internal/auth

The token service: sessions for the web app, OAuth 2 for third-party apps,
and the JWKS every service verifies access tokens with. It is the security
core, so a change here is exact or it waits.

- OAuth changes follow RFC 9700 (OAuth 2.0 Security BCP): PKCE S256 only,
  exact redirect URI matching, single-use codes, refresh token rotation with
  reuse detection. Cite the RFC section where the code enforces a rule.
- Token material (access, refresh and service tokens, codes, client
  secrets) is never logged and never in error text. Carry it as
  `config.Secret` and call `Reveal` only where it is written to the wire.
- There is no catch-all scope. A client is granted only names from
  `app.GrantableScopes`; anything else, `*` included, is refused.
- `login-codes:redeem` (`keys.ServiceScope`) is the auth service's own
  token for the user service. It is never grantable to a client, by
  registration, authorization or client credentials.
- Every route gets a row in `httpapi.TestAuthorizationMatrix` and an entry
  in `api/auth.openapi.yaml`; both tests fail on a route without one. An
  OAuth endpoint documents its errors as `OAuthError`, the rest as `Problem`.
- Every store (`memory`, `mysql`, `mongodb`) runs `store/storetest`. A new
  store rule goes into the contract suite first, then into each backend.
- `app` holds the rules and returns `apperr` kinds, or `*app.OAuthError`
  and `*app.AuthorizeError` at the OAuth endpoints; `httpapi` only maps
  them to HTTP. A store failure is always a server error.
- Tests fake the user service with `app/apptest`, a `userv1connect`
  handler; never reach a real one.

Verify: `go test -race -shuffle=on ./internal/auth/...` (add `-short`
without Docker).
