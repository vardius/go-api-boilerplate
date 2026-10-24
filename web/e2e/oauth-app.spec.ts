import { expect, test } from '@playwright/test';
import { z } from 'zod';
import { requireStack, signIn, uniqueEmail } from './support';

requireStack();

const tokenSchema = z.object({
  access_token: z.string().min(1),
  token_type: z.string().regex(/^bearer$/i),
  expires_in: z.number().positive(),
});

test('creates an OAuth app, uses its secret once shown, then deletes it', async ({ page, request }) => {
  await signIn(page, uniqueEmail());
  await page.getByRole('link', { name: 'OAuth apps' }).click();
  await page.getByRole('button', { name: 'Register an app' }).click();
  await page.getByLabel('Name').fill('E2E Notes');
  await page.getByLabel('Redirect URIs').fill('https://notes.example/callback');
  await page.getByRole('checkbox', { name: 'user:read' }).check();
  await page.getByRole('button', { name: 'Create app' }).click();

  await expect(page.getByText("Copy the client secret now. You won't be able to see it again.")).toBeVisible();
  const clientId = await page.getByRole('group', { name: 'Client ID' }).locator('code').innerText();
  const secret = await page.getByRole('group', { name: 'Client secret' }).locator('code').innerText();

  // The secret works: the client-credentials grant, authenticated with client_secret_basic (RFC 6749 §2.3.1).
  const basic = Buffer.from(`${encodeURIComponent(clientId)}:${encodeURIComponent(secret)}`).toString('base64');
  const token = await request.post('/api/auth/oauth/token', {
    headers: { Authorization: `Basic ${basic}` },
    form: { grant_type: 'client_credentials', scope: 'user:read' },
  });
  expect(token.status()).toBe(200);
  tokenSchema.parse(await token.json());

  await page.getByRole('button', { name: "I've saved the secret" }).click();
  await expect(page.getByText(secret)).toHaveCount(0);
  await expect(page.getByRole('heading', { level: 3, name: 'E2E Notes' })).toBeVisible();

  await page.getByRole('button', { name: 'Delete E2E Notes' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Delete app' }).click();

  await expect(page.getByText('E2E Notes was deleted.')).toBeVisible();
  await expect(page.getByRole('heading', { level: 3, name: 'E2E Notes' })).toHaveCount(0);
});
