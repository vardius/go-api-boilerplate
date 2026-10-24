# internal/platform

The kit both services build on. Each package has one job and a small
interface; read its package doc before using or changing it.

- Imports no service package (`internal/auth`, `internal/user`, `gen/`).
  `internal/archtest` fails the build if one does.
- A package here earns its place by hiding real work behind a small
  interface. A one-call wrapper belongs in the service that needs it.
- Every store interface (`es.Store`, `outbox.Store`) has one contract suite
  (`es/estest`, `outbox/outboxtest`) that `memdb`, `mysql` and `mongodb`
  all run. A new rule goes into the suite first, then into each backend.
- Container tests (`mysql/mysqltest`, `mongodb/mongotest`) skip under
  `-short`; CI runs them. Keep testcontainers out of non-test packages.
- Config structs use unprefixed env names with a `desc` tag on every field;
  services add the prefix.

Verify: `go test -race ./internal/platform/...` (add `-short` without Docker).
