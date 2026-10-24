---
name: add-endpoint
description: Add or change an HTTP route on the user or auth service, keeping the OpenAPI document, the authorization matrix, the web app's schemas and the docs in step. Use whenever a route, status code or JSON field changes.
---

# Add an endpoint

Routes, their documents and their callers change together, in one change.
The [add an endpoint guide](https://vardius.github.io/go-api-boilerplate/guides/add-an-endpoint/)
explains why; this is the checklist. `<svc>` is `user` or `auth`.

## Steps

1. **Use case first.** The handler only decodes, calls one `app` method and
   encodes. Scopes and ownership are checked in `app`, from the identity in
   the context, never from the body or the path.
2. **Route** in `internal/<svc>/httpapi/httpapi.go`, on the gorouter. Wrap
   the handler in `httpx.Handle` so errors become problem details through the
   one table in `httpx`. Decode bodies with `httpx.DecodeJSON` and pages with
   `httpx.ParsePage`. A route that reads a cookie and changes state also
   needs `httpx.CrossOrigin`; a public write needs a rate limit.
3. **OpenAPI** in `api/<svc>.openapi.yaml`: the path, every status the
   handler can return, and the schemas. The tests fail when a route and the
   document disagree, and every response a test sees is validated against
   it.
4. **Authorization matrix**: add the route's row to
   `internal/<svc>/httpapi/matrix_test.go`, with the status for each kind of
   caller. A route without a row fails the test.
5. **Handler tests** in `httpapi`: success, each error status, and the
   response shape.
6. **Web app**, if it calls the route: the zod schema in
   `web/src/api/schemas.ts`, the call in `web/src/api/endpoints.ts`, and
   every message in both `web/src/i18n/en.ts` and `pl.ts`.
7. **e2e**, for a user-visible flow: a case in `e2e/e2e_test.go`.
8. **Docs**: the website page for the flow. The REST reference is generated
   from `api/*.openapi.yaml`.

## Verify

```bash
go test -race -shuffle=on ./internal/<svc>/...
make check
```
