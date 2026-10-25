# internal/platform/es

Event sourcing. Stored events are kept forever, so:

- Events are immutable facts, named in the past tense. Never edit or delete
  a stored event to fix data: append a correcting event.
- A registered type name (`user.registered.v1`) never changes, and neither
  does the shape stored under it. A new shape gets a new name (`.v2`) and a
  `Registry.Upcast` from the old one.
- Events hold no secrets (tokens, passwords, codes) and no personal data
  beyond what the fact needs. `Meta` holds identifiers only.
- `Apply` only assigns. Rules are checked before `Raise`.
- Projections run in the append transaction and must be idempotent: running
  one twice on the same events leaves the same rows.
- A concurrent writer is an `es.ErrConcurrency`, which reaches clients as a
  409. Don't retry it inside a store.
- Stream IDs are plain UUIDs and say nothing about where a stream lives.
  If streams ever need one owner per group (a sharded processor, a
  partitioned consumer), route by a hash of the stream ID, never by bits
  added to new IDs: the old IDs live in stored events forever.
- A read model kept outside the append transaction (in memory, in a search
  index) is fed only by events, applies them idempotently, and can be
  thrown away and rebuilt. Rebuilding needs an ordered feed of every
  stream with a checkpoint, which `Store` doesn't have: add it, with its
  rules in `estest`, before such a read model.
