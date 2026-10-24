import { expect, test } from '@playwright/test';
import { requireStack, signIn, uniqueEmail } from './support';

requireStack();

const iphone =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 19_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/19.0 Mobile/15E148 Safari/604.1';

test('revokes the session on another device', async ({ page, browser }) => {
  const email = uniqueEmail();
  const phoneContext = await browser.newContext({ userAgent: iphone });
  const phone = await phoneContext.newPage();
  await signIn(page, email);
  await signIn(phone, email);

  await page.getByRole('link', { name: 'Devices' }).click();
  const devices = page.getByRole('region', { name: 'Signed-in devices' }).getByRole('listitem');
  await expect(devices).toHaveCount(2);
  await page.getByRole('button', { name: 'Revoke Safari on iOS' }).click();

  await expect(devices).toHaveCount(1);
  await expect(devices.getByText('This device')).toBeVisible();
  // The access token may live out its 15 minutes, but the revoked session can no longer be renewed.
  const refresh = await phone.request.post('/api/auth/v1/sessions/refresh');
  expect(refresh.status()).toBe(401);
  await phoneContext.close();
});
