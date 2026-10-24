import { defineConfig, devices } from '@playwright/test';

/**
 * End-to-end specs against the running stack (`docker compose up --wait` at
 * the repo root). They skip unless E2E_STACK_UP=1 says the stack is up.
 */
export default defineConfig({
  testDir: 'e2e',
  // The APIs rate-limit sign-in links per client, so one worker keeps runs predictable.
  workers: 1,
  forbidOnly: process.env.CI !== undefined,
  retries: process.env.CI === undefined ? 0 : 1,
  reporter: process.env.CI === undefined ? 'list' : 'github',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:8080',
    locale: 'en-US',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
