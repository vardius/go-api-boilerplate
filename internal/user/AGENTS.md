# internal/user

The user service: an event-sourced `User`, sign-in links, email change and
Google sign-in. `cmd/user` wires it; read the package doc of each part first.

| Package | Job |
| --- | --- |
| `domain` | the `User` aggregate, its events, `Email`; stdlib, `es` and `apperr` only |
| `app` | one method per use case; each write is one `uow.Do` |
| `store` | the `users` projection and login codes: one contract (`storetest`), three backends |
| `mail` | `Mailer`, SMTP, templates and the outbox handlers |
| `httpapi` | the public routes and Google OIDC |
| `rpc` | `user.v1.LoginService` for the auth service |

- The caller is the `authn.Identity` in the context (`app.Actor`), never a
  field of the body or the path. Scopes are checked in `app`, so HTTP and
  Connect can't disagree.
- Every route gets a row in `httpapi/matrix_test.go` and an entry in
  `api/user.openapi.yaml`, in the same change. The tests fail on a route
  without either, and every response a test sees is checked against the
  document.
- Events are immutable once released. A new shape is a new `.vN` type with
  an upcaster from the old one; never edit a released event's fields or
  JSON names.
- A store rule goes into `store/storetest` first, then into memory, MySQL
  and MongoDB. Reads run outside transactions and writes take the `uow.Tx`;
  the function given to `uow.Do` only writes, since MongoDB may run it
  again.
- Nothing tells a caller whether an address has an account:
  `RequestLoginLink` succeeds either way, and a taken address shows only
  when an email change is confirmed.
- `login_codes` keeps only a code's SHA-256. The code itself is only in
  the outbox message that mails it (removed on delivery, kept if buried)
  and never in a log.

Verify: `go test -race -shuffle=on ./internal/user/...` (add `-short`
without Docker).
