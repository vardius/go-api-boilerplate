# web

The account app: sign in by email link, profile, devices, OAuth apps and the
OAuth consent screen. Vite, React and TypeScript, served by nginx in front of
the auth and user services.

## Develop

```sh
npm ci && npm run dev
```

The dev server runs on http://localhost:5173 and proxies `/api` to
http://localhost:8080, so start the stack first (`docker compose up --wait`
from the repo root). In compose, sign-in emails land in Mailpit at
http://localhost:8025.

## Test

```sh
npm run check   # typecheck, lint and unit tests
npm run build
```

The Playwright specs in `e2e/` drive the running stack and read sign-in links
from Mailpit's API. They skip unless `E2E_STACK_UP=1`:

```sh
npx playwright install chromium
E2E_STACK_UP=1 npm run e2e
```

`E2E_BASE_URL` (default http://localhost:8080) and `MAILPIT_URL` (default
http://localhost:8025) point them elsewhere.
