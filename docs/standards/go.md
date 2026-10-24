# Standards: Go

On top of [shared.md](shared.md).

## Write

- Accept interfaces, return structs. Define small interfaces where they are
  consumed, not where they are implemented, and not before a second
  implementation needs one.
- `context.Context` is the first parameter of anything that does I/O or can
  block. Pass it through; never `context.Background()` mid-stack.
- Wrap every returned error with context: `fmt.Errorf("load user %s: %w", id, err)`.
  Compare with `errors.Is` and `errors.AsType`, never by string. Errors carry
  one of the `apperr` kinds, and only `apperr.New` or `*apperr.ValidationError`
  text reaches a client.
- Table-driven tests; `-race` and `-shuffle=on` in CI; `t.Context()` in tests;
  `testing/synctest` for anything with timers.
- Package by domain, not by layer. No `utils` or `common` packages. Unexport
  what has no outside consumer; a method that ignores its receiver is a
  function.
- Prefer the standard library: `slices`, `maps`, `cmp.Or`, `strings.Cut`,
  the `uuid` package, `log/slog`, `errors.AsType`, `sync.WaitGroup.Go`.
  `go fix ./...` leaves the tree unchanged.
- Sets are `map[K]struct{}`. Mutually exclusive states are one enum type,
  not several bools. More than one bool parameter becomes an options struct.
- Inject the clock (`Now func() time.Time`) wherever logic reads the time.
- Compile regular expressions once, at package level. Cleanup goes in
  `defer`.
- Independent concurrent work uses `errgroup` with `SetLimit`.

## Bug patterns to hunt

- `err` shadowed by `:=` in an inner scope, checked against the wrong
  variable, or discarded with `_`.
- Goroutine leaks: no exit on `ctx.Done()`, a send nobody receives, a missing
  `Wait`.
- Data races: a map written from two goroutines, a struct holding a mutex
  copied by value.
- A type assertion without `, ok`. Inside a goroutine it takes the process
  down.
- `defer` in a loop; `rows.Close()` missing; `rows.Err()` or `cur.Err()`
  unchecked.
- Transactions: no rollback on an error path; work outside the transaction
  that belongs inside it; a read that assumes isolation the database doesn't
  give.
- Nil map writes; a nil pointer from an error path's zero value; a typed nil
  interface comparing `!= nil`.
- `time.Time` compared with `==`.
- HTTP: a response body not closed; a client without a timeout; the status
  unchecked before decoding.
- JSON and BSON: `omitempty` on a bool or number that must be written;
  numbers decoded into `float64`; unexported fields skipped silently.
- SQL and documents: a query per item where one batch would do; a new query
  shape without an index; `NULL` scanned into a non-pointer.
- Leases and retries: a claimed item waiting on a lock while its lease runs
  out; a "done" marker written before the side effect it guards.
- `append` aliasing a shared backing array.
- A variable shadowing a package name (`uuid := uuid.New()`).
