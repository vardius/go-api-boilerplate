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
   `httpx.ParsePage`. A public write needs a rate limit.
   - **user**: a route with a caller is wrapped in `identify`; without it
     `app.Actor` sees no identity and every call is a 401. The app checks
     the scope. Cross-origin checks are router middleware already.
   - **auth**: wrap the route in `identify` plus `authn.RequireUser(scope)`,
     or `firstParty` for the web app's own routes, and add `cookie`
     (`httpx.CrossOrigin`) to one that reads a cookie and changes state.
3. **OpenAPI** in `api/<svc>.openapi.yaml`: the path, every status the
   handler can return, and the schemas. The tests fail when a route and the
   document disagree, and every response a test sees is validated against
   it.
4. **Authorization matrix**: add the route's row to
   `internal/<svc>/httpapi/matrix_test.go`, with the status for each caller
   that file lists. A route without a row fails the test. On the user
   service, a route that answers with the caller's user also goes in the
   list of patterns whose 200 body is checked to be the caller's.
5. **Handler tests** in `httpapi`: success, each error status, and the
   response shape.
6. **Web app**, if it calls the route: the zod schema in
   `web/src/api/schemas.ts`, the call in `web/src/api/endpoints.ts` (the
   `method` union in `web/src/api/client.ts` lists the verbs in use; widen
   it for a new one), the query or cache update in `web/src/api/queries.ts`,
   the screen and its tests in `web/src/screens`, and every message in both
   `web/src/i18n/en.ts` and `pl.ts`. Check the screen at 320 px with 1.3×
   text, as `web/AGENTS.md` says.
7. **e2e**, for a user-visible flow: a case in `e2e/e2e_test.go`.
   `make check` vets it; it runs against a stack (`make up`, then
   `make e2e`), and in CI for every store.
8. **Docs**: the website page for the flow and the REST reference, which
   is built from `api/*.openapi.yaml`.

Without Docker, finish the rest, say that e2e and the 320 px check didn't
run, and let CI run e2e.

## Verify

```bash
go test -race -shuffle=on ./internal/<svc>/...
make check
```
