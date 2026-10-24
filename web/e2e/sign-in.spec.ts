import { expect, test } from '@playwright/test';
import { mailedLink, requireStack, signIn, uniqueEmail } from './support';

requireStack();

test('signs in with the link from the email', async ({ page }) => {
  const email = uniqueEmail();

  await signIn(page, email);

  await expect(page.getByText(email, { exact: true })).toBeVisible();
});

test('a sign-in link works only once', async ({ page, browser }) => {
  const email = uniqueEmail();
  await page.goto('/sign-in');
  await page.getByLabel('Email').fill(email);
  await page.getByRole('button', { name: 'Email me a sign-in link' }).click();
  const link = await mailedLink(page.request, email, '/sign-in/complete');
  await page.goto(link);
  await expect(page.getByRole('heading', { level: 1, name: 'Profile' })).toBeVisible();

  // Another browser didn't ask for the link, so it asks before using it.
  const elsewhere = await browser.newContext();
  const replay = await elsewhere.newPage();
  await replay.goto(link);
  await expect(replay.getByRole('heading', { level: 1, name: 'Sign in with this link?' })).toBeVisible();
  await replay.getByRole('button', { name: 'Sign in with this link' }).click();

  await expect(replay.getByRole('heading', { level: 1, name: 'This sign-in link has expired' })).toBeVisible();
  await elsewhere.close();
});
