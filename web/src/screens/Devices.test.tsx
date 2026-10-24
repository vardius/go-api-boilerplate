import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { empty, json, mockApi, problem } from '../test/api';
import { renderApp } from '../test/render';

const list = 'GET /api/auth/v1/sessions';
const mac =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36';
const iphone =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 19_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/19.0 Mobile/15E148 Safari/604.1';

const current = {
  id: 's1',
  created_at: '2026-10-01T08:00:00Z',
  last_used_at: '2026-10-24T08:00:00Z',
  user_agent: mac,
  current: true,
};
const phone = {
  id: 's2',
  created_at: '2026-09-01T08:00:00Z',
  last_used_at: null,
  user_agent: iphone,
  current: false,
};

describe('Devices', () => {
  it('lists sessions with this device first and marked', async () => {
    mockApi({ [list]: json(200, { sessions: [phone, current] }) });
    renderApp('/devices');

    const region = await screen.findByRole('region', { name: 'Signed-in devices' });
    const items = within(region).getAllByRole('listitem');
    expect(items.map((item) => within(item).getByRole('heading').textContent)).toEqual([
      'Chrome on macOS',
      'Safari on iOS',
    ]);
    expect(items.map((item) => within(item).queryByText('This device') !== null)).toEqual([true, false]);
  });

  it('revokes another session and then says there are no others to sign out of', async () => {
    const api = mockApi({
      [list]: [json(200, { sessions: [current, phone] }), json(200, { sessions: [current] })],
      'DELETE /api/auth/v1/sessions/s2': empty(204),
    });
    const { user } = renderApp('/devices');

    await user.click(await screen.findByRole('button', { name: 'Revoke Safari on iOS' }));

    expect((await screen.findByText('Safari on iOS is signed out.')).closest('[role="status"]')).toHaveFocus();
    const none = await screen.findByRole('button', { name: 'Sign out of other devices' });
    expect(none).toHaveAttribute('aria-disabled', 'true');
    expect(none).toHaveAccessibleDescription("You aren't signed in on any other device.");
    expect(api.callsTo('DELETE /api/auth/v1/sessions/s2')).toHaveLength(1);
  });

  it('signs out of every other device at once', async () => {
    const tablet = { ...phone, id: 's3' };
    const api = mockApi({
      [list]: [json(200, { sessions: [current, phone, tablet] }), json(200, { sessions: [current] })],
      'DELETE /api/auth/v1/sessions/s2': empty(204),
      // Already gone elsewhere: still the outcome asked for.
      'DELETE /api/auth/v1/sessions/s3': problem(404),
    });
    const { user } = renderApp('/devices');

    await user.click(await screen.findByRole('button', { name: 'Sign out of 2 other devices' }));

    expect(await screen.findByRole('button', { name: 'Sign out of other devices' })).toBeInTheDocument();
    expect(screen.getByText('Signed out of 2 other devices.').closest('[role="status"]')).toHaveFocus();
    expect(api.callsTo('DELETE /api/auth/v1/sessions/s2')).toHaveLength(1);
    expect(api.callsTo('DELETE /api/auth/v1/sessions/s3')).toHaveLength(1);
  });

  it('signs this device out and lands on sign-in', async () => {
    mockApi({
      [list]: json(200, { sessions: [current] }),
      'DELETE /api/auth/v1/sessions/current': empty(204),
      'GET /api/user/v1/config': json(200, { google_enabled: false }),
    });
    const { router, user } = renderApp('/devices');

    await user.click(await screen.findByRole('button', { name: 'Sign out' }));

    expect(await screen.findByText("You're signed out.")).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/sign-in');
  });
});
