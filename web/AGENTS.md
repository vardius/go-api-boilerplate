# web: rules for agents

The code bar is [`docs/standards/typescript.md`](../docs/standards/typescript.md); these add to it.

## Commands

- `npm run check`: typecheck, lint and unit tests. It must pass before a commit.
- `npm run build`: the production bundle in `dist/`.
- `E2E_STACK_UP=1 npm run e2e`: Playwright against the running compose stack.

## Auth and the API

- Sessions live in HttpOnly cookies. JS never reads, writes or stores a token:
  no `document.cookie`, no tokens in storage (lint bans `document.cookie`).
- Every request goes through `src/api/client.ts` (lint bans `fetch` elsewhere):
  same-origin, one refresh on 401, problem+json parsed into `ProblemError`.
- Validate every response with zod (`src/api/schemas.ts`); never cast one.
- A network error offers a retry; it never signs the reader out.
- `next` and other redirect targets go through `safeNext` (same origin only).

## UI

- Flawless at 320 px wide with 1.3× text: nothing leaves its box. Check new
  screens there, in light and dark.
- Exactly one primary action per screen.
- A disabled or locked control says why and what to do (`aria-disabled` plus
  `aria-describedby`, not a silent `disabled`).
- Every control has a visible label; focus is always visible; colours come
  from `src/styles/tokens.css` (AA in both themes); sizes in rem; motion
  respects `prefers-reduced-motion`.
- When an action removes the focused control, move focus somewhere meaningful
  (the result, the next heading or the control that opened it).

## Strings

- Every user-visible string is a message in both `src/i18n/en.ts` and
  `src/i18n/pl.ts`. Polish plurals need one, few, many and other;
  `messages.test.ts` checks that both locales match.
- Never invent product facts in copy (lifetimes, env vars, limits): take them
  from the server or the website's reference pages.
