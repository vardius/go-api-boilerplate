---
name: add-command
description: Add a state change to the event-sourced user service (a new event, aggregate method and use case), such as "let users set a display name". Use for any write that changes a User.
---

# Add a command

A command changes a `User` by raising an event. The event is stored, the
`users` projection follows it in the same transaction, and anything the
outside world must hear about goes through the outbox. The full walkthrough
is the [add a command guide](https://vardius.github.io/go-api-boilerplate/guides/add-a-command/);
this is the checklist.

Read `internal/user/AGENTS.md` and `internal/platform/es/AGENTS.md` first.

## Steps, in this order

1. **Event** in `internal/user/domain/events.go`: a struct with JSON tags,
   registered in `RegisterEvents` as `user.<past_tense>.v1`. Never reuse or
   rename a released name; a changed shape is a `.v2` with an upcaster
   (`Registry.Upcast`). No secrets in events.
2. **Aggregate method** in `domain/user.go`: check the invariants, return a
   `domain` error (an `apperr` kind) when one fails, else `es.Raise` the
   event. Handle the event in `Apply`; `Apply` never fails on a valid
   history. The domain imports only the standard library, `platform/es` and
   `platform/apperr` (`internal/archtest` fails otherwise).
3. **Domain tests** in `domain/user_test.go` with `estest.Given(...).When(...).Then(...)`
   or `.ThenError(...)`: the event raised, and every invariant refused.
4. **Projection**: if the event changes a column of `store.User`, add the
   case to `Project` in `store/memory`, `store/mysql` and `store/mongodb`.
   A new column means a goose migration in `store/mysql/migrations` and, on
   MongoDB, no `omitempty` on a field that must be written. Write the rule
   into `store/storetest` first; all three backends run it.
5. **Use case** in `internal/user/app/app.go`: a method on `Service` that
   gets the caller from `Actor(ctx, scope)`, which also checks the scope
   (never from the body), then does the whole write in one `s.uow.Do`: load, call the aggregate method,
   `s.repo.Save`, and any `outbox.NewMessage`. The function given to `Do`
   only writes, because MongoDB may run it again. Test it in `app_test.go`
   with the memory stores and an injected clock.
6. **Endpoint**: follow the add-endpoint skill (route, OpenAPI entry,
   authorization matrix row, web schema).
7. **Docs**: the website page that describes the flow, if a person would
   notice the change.

## Verify

```bash
go test -race -shuffle=on ./internal/user/... ./internal/archtest/...
make check
```

Without Docker, add `-short`; CI runs the MySQL and MongoDB store tests.
