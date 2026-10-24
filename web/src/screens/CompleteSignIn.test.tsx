import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { loadPage } from '../lib/browser';
import { json, mockApi, offline, problem } from '../test/api';
import { apiUser, renderApp } from '../test/render';

vi.mock('../lib/browser', () => ({ loadPage: vi.fn() }));

const sessions = 'POST /api/auth/v1/sessions';
const me = 'GET /api/user/v1/me';
const refresh = 'POST /api/auth/v1/sessions/refresh';
const created = json(201, { session: { id: 's1', expires_at: '2026-11-23T10:00:00Z' } });
/** A browser with no session: /me is 401 and so is the refresh it tries. */
const signedOut = { [me]: problem(401), [refresh]: problem(401) };

describe('Complete sign-in', () => {
  it('with no session, trades the code for a session at once, then goes to next', async () => {
    const api = mockApi({
      ...signedOut,
      [sessions]: created,
      'GET /api/auth/v1/clients': json(200, { clients: [], limit: 20, offset: 0 }),
    });
    const { router } = renderApp('/sign-in/complete?code=abc&next=%2Fapps');

    expect(await screen.findByRole('heading', { level: 1, name: 'OAuth apps' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/apps');
    expect(api.callsTo(me)).toHaveLength(1);
    expect(api.callsTo(sessions)).toEqual([expect.objectContaining({ body: { code: 'abc' } })]);
  });

  it('with a session, asks before switching and posts nothing until the reader confirms', async () => {
    const api = mockApi({
      [me]: json(200, apiUser({ email: 'ada@example.com' })),
      [sessions]: created,
      'GET /api/auth/v1/clients': json(200, { clients: [], limit: 20, offset: 0 }),
    });
    const { user, router } = renderApp('/sign-in/complete?code=abc&next=%2Fapps');

    expect(await screen.findByRole('heading', { level: 1, name: 'Sign in to a different account?' })).toBeInTheDocument();
    expect(screen.getByText('ada@example.com')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Stay signed in as ada@example.com' })).toHaveAttribute('href', '/apps');
    expect(api.callsTo(sessions)).toHaveLength(0);

    await user.click(screen.getByRole('button', { name: 'Sign in with this link' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'OAuth apps' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/apps');
    expect(api.callsTo(sessions)).toEqual([expect.objectContaining({ body: { code: 'abc' } })]);
  });

  it('with a session, staying signed in goes on without using the code', async () => {
    const api = mockApi({
      [me]: json(200, apiUser()),
      'GET /api/user/v1/config': json(200, { google_enabled: false }),
    });
    const { user, router } = renderApp('/sign-in/complete?code=abc');

    await user.click(await screen.findByRole('link', { name: 'Stay signed in as ada@example.com' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Profile' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/profile');
    expect(api.callsTo(sessions)).toHaveLength(0);
  });

  it("doesn't use the code while it can't tell whether someone is signed in", async () => {
    const api = mockApi({ [me]: [offline(), problem(401)], [refresh]: problem(401), [sessions]: created });
    const { user } = renderApp('/sign-in/complete?code=abc&next=%2Fapps');

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
    expect(api.callsTo(sessions)).toHaveLength(0);

    api.on('GET /api/auth/v1/clients', json(200, { clients: [], limit: 20, offset: 0 }));
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'OAuth apps' })).toBeInTheDocument();
    expect(api.callsTo(sessions)).toHaveLength(1);
  });

  it('goes to the profile when there is no next', async () => {
    mockApi({
      [me]: [problem(401), json(200, apiUser())],
      [refresh]: problem(401),
      [sessions]: created,
      'GET /api/user/v1/config': json(200, { google_enabled: false }),
    });
    const { router } = renderApp('/sign-in/complete?code=abc');

    expect(await screen.findByRole('heading', { level: 1, name: 'Profile' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/profile');
  });

  it('loads a server route such as the OAuth authorize endpoint as a full page', async () => {
    mockApi({ ...signedOut, [sessions]: created });
    const next = '/api/auth/oauth/authorize?client_id=c1&state=xyz';
    renderApp(`/sign-in/complete?code=abc&next=${encodeURIComponent(next)}`);

    await waitFor(() => {
      expect(loadPage).toHaveBeenCalledWith(next);
    });
  });

  it('explains an expired or used code and offers a new link that keeps next', async () => {
    mockApi({ ...signedOut, [sessions]: problem(401) });
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
      [me]: [problem(401), json(200, apiUser())],
      [refresh]: problem(401),
      [sessions]: [offline(), created],
      'GET /api/user/v1/config': json(200, { google_enabled: false }),
    });
    const { user } = renderApp('/sign-in/complete?code=abc');

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Profile' })).toBeInTheDocument();
  });
});
