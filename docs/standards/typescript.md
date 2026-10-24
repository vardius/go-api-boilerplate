# Standards: TypeScript and React

On top of [shared.md](shared.md).

## Write

- Strict TypeScript: no `any`, no non-null `!` to quiet the compiler. Parse
  every external value (responses, URL params, storage) with zod at the
  boundary.
- Derive values during render; keep state only for what can't be derived.
  Don't mirror props into state.
- Effects synchronise with outside systems; they aren't data flow between
  components. Every effect that subscribes also cleans up.
- Server state lives in TanStack Query, not in component state.
- Compose (children, slots) instead of piling up boolean props.
- React 19: `ref` is a plain prop, context is read with `use(Context)`, and
  components are plain functions.
- Style with the design tokens and CSS modules; no one-off colours or sizes.
  Sizes in `rem`, so text scaling works.
- Accessible by default: native elements over clickable `div`s, a label for
  every control, visible focus, focus moved and restored around dialogs.
- Every user-visible string goes through react-intl, in every locale.
- Test what the user sees (Testing Library), not internals.

## Bug patterns to hunt

- Async races: a slow earlier response overwriting a newer one; no abort
  signal; state set after unmount.
- Stale closures: callbacks, intervals and listeners reading old state;
  wrong effect or memo dependencies.
- Keys: index keys on lists that reorder, duplicate keys.
- `||` where `??` was meant (`0`, `''`, `false`); `count && <X />` rendering
  `0`.
- Loading, empty and error collapsed into one branch, so the screen says
  "nothing here" while it is still loading.
- Optimistic updates never rolled back on failure.
- An input switching between controlled and uncontrolled.
- Dates: `new Date(string)` parsing, local versus UTC.
- `localStorage` and `JSON.parse` throwing in private mode or on a corrupt
  value.
- Unhandled promise rejections; `await` in a loop where parallel was meant.
- `str.replace(x, replacement)` expanding `$&` and `$$`: pass a function
  for literal text.
- An open redirect: a `next` parameter followed without checking that it is
  a same-origin relative path.
