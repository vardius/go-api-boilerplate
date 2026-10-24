# Standards: every change

The bar a change meets before review, whoever or whatever wrote it. Each
rule is something a reviewer can check. The `AGENTS.md` nearest the code adds
rules for its folder and wins where it is stricter. Stack rules:
[go.md](go.md), [typescript.md](typescript.md).

## Before writing

- Read the area first: the files you'll touch, their callers, their tests,
  and the `AGENTS.md` files on the path to them.
- Search before you create. A helper, type, fixture or component that
  already exists is reused or extended, never rewritten beside the original.
- Never invent an environment variable, endpoint, flag or config key. Find
  it in the code or the docs, or say it's unknown.
- Match local conventions: naming, layout, error handling, test style.
- List the failure modes (below) before the code. Each becomes a test or a
  stated decision.

## Design

- The smallest change that fully solves the problem. No speculative
  options, flags, layers, or handling for states the system can't produce.
- One source of truth per fact. Derive; don't copy state that can drift.
- Every piece of state has one owner, and writes go through it. When a guard
  reads state your own write changes, the write path updates that state too.
- Make illegal states unrepresentable. Keep transient states (loading,
  pending) apart from terminal ones (missing, empty, failed); merging them is
  the most common bug a fix introduces.
- Validate at the boundary (HTTP, RPC, disk, environment); trust typed
  values inside.
- One responsibility per unit: if its name needs "and", split it. Prefer
  deep modules, a small interface with real work behind it, to pass-through
  layers and one-call helpers.
- Hot paths (per request, per event, per render) get explicit cost
  reasoning: no O(n²) on unbounded input, no I/O in loops.

## Code

- Names say what a thing does, in domain terms. A comment explains a *why*
  that no name or test can hold; it never narrates the code.
- Return early: handle the failure first and keep the success path flat.
- Errors are handled or returned with context; never swallowed, never
  logged and ignored without a stated reason.
- Delete dead code, commented-out code and compatibility shims the change
  makes unnecessary. Git is the archive.
- Persisted, serialized and cross-service shapes (events, proto, OpenAPI,
  the web app's schemas) change on every side in the same change, and
  decoding tolerates what an older writer stored.
- Secrets live only in gitignored env files or a secret store. Templates
  hold placeholders, and code never falls back to a real value.
- Every sentence of documentation you add is true of the code.

## Changing existing code

- A refactor keeps every branch, error path and side effect. An `if/else`
  that became an `if` dropped behaviour: find where it went.
- Before removing a field, parameter or path, find why it exists and check
  every caller.
- After moving or removing a symbol, search the whole tree: docs, configs,
  scripts and the website, not just source files.

## Tests

- Test behaviour through the public interface, not the implementation. A
  test that breaks on a refactor that kept behaviour was testing internals.
- Expected values come from an independent source (a literal, a worked
  example, the spec), never recomputed the way the code computes them.
- Mock only at system boundaries: network, time, randomness, an external
  service. Your own modules and the in-memory store run for real.
- Every bug fix has a regression test that fails before the fix.
- A suite earns trust when a wrong constant, a flipped branch or a missing
  side effect makes some test fail. Try one such mutation per change.
- Deterministic: injected clocks, no sleeps, no real network. Time-dependent
  tests also pass under `TZ=Pacific/Kiritimati`.

## Failure modes to list

- **Inputs:** empty, missing, zero, negative, huge, duplicate, unicode,
  malformed.
- **Time:** time zones, day boundaries, clock skew, ordering of timestamps.
- **State:** first run, already done (idempotency), partly done, stale
  cache, data written by an older version, absent versus zero.
- **Concurrency:** two callers at once, interleaving, cancellation, retries,
  out-of-order completion.
- **Failure:** network, disk and permission errors, timeouts, partial
  writes.
- **Callers:** every existing caller of what changed, and behaviour the diff
  deletes.

When a bug gets past review, make the tooling catch the next one: a
mechanical pattern becomes a lint rule or a test; a judgement call becomes a
line in the matching stack file.
