import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { loadPage } from '../lib/browser';
import { json, mockApi, offline, problem } from '../test/api';
import { apiUser, renderApp } from '../test/render';

vi.mock('../lib/browser', () => ({ loadPage: vi.fn() }));

const sessions = 'POST /api/auth/v1/sessions';
const created = json(201, { session: { id: 's1', expires_at: '2026-11-23T10:00:00Z' } });

describe('Complete sign-in', () => {
  it('trades the code for a session once, then goes to next', async () => {
    const api = mockApi({
      [sessions]: created,
      'GET /api/auth/v1/clients': json(200, { clients: [], limit: 20, offset: 0 }),
    });
    const { router } = renderApp('/sign-in/complete?code=abc&next=%2Fapps');

    expect(await screen.findByRole('heading', { level: 1, name: 'OAuth apps' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/apps');
    expect(api.callsTo(sessions)).toEqual([expect.objectContaining({ body: { code: 'abc' } })]);
  });

  it('goes to the profile when there is no next', async () => {
    mockApi({
      [sessions]: created,
      'GET /api/user/v1/me': json(200, apiUser()),
      'GET /api/user/v1/config': json(200, { google_enabled: false }),
    });
    const { router } = renderApp('/sign-in/complete?code=abc');

    expect(await screen.findByRole('heading', { level: 1, name: 'Profile' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/profile');
  });

  it('loads a server route such as the OAuth authorize endpoint as a full page', async () => {
    mockApi({ [sessions]: created });
    const next = '/api/auth/oauth/authorize?client_id=c1&state=xyz';
    renderApp(`/sign-in/complete?code=abc&next=${encodeURIComponent(next)}`);

    await waitFor(() => {
      expect(loadPage).toHaveBeenCalledWith(next);
    });
  });

  it('explains an expired or used code and offers a new link that keeps next', async () => {
    mockApi({ [sessions]: problem(401) });
    renderApp('/sign-in/complete?code=old&next=%2Fapps');

    expect(await screen.findByRole('heading', { name: 'This sign-in link has expired' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Send me a new link' })).toHaveAttribute('href', '/sign-in?next=%2Fapps');
  });

  it('says when the link has no code', () => {
    const api = mockApi();
    renderApp('/sign-in/complete');

    expect(screen.getByRole('heading', { name: 'This sign-in link is incomplete' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Send me a new link' })).toHaveAttribute('href', '/sign-in');
    expect(api.calls).toHaveLength(0);
  });

  it('offers a retry when the server is unreachable', async () => {
    mockApi({
      [sessions]: [offline(), created],
      'GET /api/user/v1/me': json(200, apiUser()),
      'GET /api/user/v1/config': json(200, { google_enabled: false }),
    });
    const { user } = renderApp('/sign-in/complete?code=abc');

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Profile' })).toBeInTheDocument();
  });
});
