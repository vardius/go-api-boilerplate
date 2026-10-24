import { randomUUID } from 'node:crypto';
import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import { z } from 'zod';

/** Skips the calling spec file unless the stack is known to be running. */
export function requireStack(): void {
  test.skip(
    process.env.E2E_STACK_UP !== '1',
    'Needs the running stack: `docker compose up --wait`, then E2E_STACK_UP=1 npm run e2e.',
  );
}

const mailpit = process.env.MAILPIT_URL ?? 'http://localhost:8025';

const searchSchema = z.object({ messages: z.array(z.object({ ID: z.string() })) });
const messageSchema = z.object({ Text: z.string() });

/** An address no other run has used, so its inbox holds only this test's mail. */
export function uniqueEmail(): string {
  return `e2e-${randomUUID()}@example.com`;
}

/**
 * Waits for mail to `email` containing a link to `path`, and returns that
 * link as a path on the app's origin. The message is then deleted, so the
 * next call waits for new mail. Uses Mailpit's API:
 * GET /api/v1/search, GET /api/v1/message/{ID}, DELETE /api/v1/search.
 */
export async function mailedLink(request: APIRequestContext, email: string, path: string): Promise<string> {
  const query = `to:"${email}"`;
  let link: string | undefined;
  await expect
    .poll(
      async () => {
        const search = await request.get(`${mailpit}/api/v1/search`, { params: { query, limit: 1 } });
        if (!search.ok()) return undefined;
        const [latest] = searchSchema.parse(await search.json()).messages;
        if (latest === undefined) return undefined;
        const message = await request.get(`${mailpit}/api/v1/message/${latest.ID}`);
        const { Text } = messageSchema.parse(await message.json());
        const found = Text.match(/https?:\/\/[^\s"'<>]+/g)?.find((url) => new URL(url).pathname === path);
        link = found === undefined ? undefined : new URL(found).pathname + new URL(found).search;
        return link;
      },
      { message: `mail to ${email} with a link to ${path}`, timeout: 20_000 },
    )
    .toBeDefined();
  await request.delete(`${mailpit}/api/v1/search`, { params: { query } });
  if (link === undefined) throw new Error(`No link to ${path} in mail to ${email}.`);
  return link;
}

/** Signs in through the emailed link, as a person would, and waits for the profile. */
export async function signIn(page: Page, email: string): Promise<void> {
  await page.goto('/sign-in');
  await page.getByLabel('Email').fill(email);
  await page.getByRole('button', { name: 'Email me a sign-in link' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Check your inbox' })).toBeVisible();

  await page.goto(await mailedLink(page.request, email, '/sign-in/complete'));
  await expect(page.getByRole('heading', { level: 1, name: 'Profile' })).toBeVisible();
}
