import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { json, mockApi, offline, problem } from '../test/api';
import { apiUser, renderApp } from '../test/render';

const me = 'GET /api/user/v1/me';
const config = 'GET /api/user/v1/config';
const emailChange = 'POST /api/user/v1/me/email-change';

describe('Profile', () => {
  it('shows the email and, with Google off, a disabled link control that says why and how to enable it', async () => {
    mockApi({ [me]: json(200, apiUser()), [config]: json(200, { google_enabled: false }) });
    renderApp('/profile');

    expect(await screen.findByText('ada@example.com')).toBeInTheDocument();
    expect(screen.getByText('Member since January 15, 2026')).toBeInTheDocument();
    const link = await screen.findByRole('button', { name: 'Link Google account' });
    expect(link).toHaveAttribute('aria-disabled', 'true');
    expect(link).toHaveAccessibleDescription(/GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET/);
  });

  it('links to Google sign-in when the server has it', async () => {
    mockApi({ [me]: json(200, apiUser()), [config]: json(200, { google_enabled: true }) });
    renderApp('/profile');

    expect(await screen.findByRole('link', { name: 'Link Google account' })).toHaveAttribute(
      'href',
      '/api/user/v1/google/start?link=1',
    );
  });

  it('confirms a just-linked Google account', async () => {
    mockApi({ [me]: json(200, apiUser({ google_linked: true })), [config]: json(200, { google_enabled: true }) });
    renderApp('/profile?google=linked');

    expect(screen.getByText('Google account linked.')).toBeInTheDocument();
    expect(await screen.findByText('Linked. You can sign in with Google.')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Link Google account' })).not.toBeInTheDocument();
  });

  it('sends a confirmation link to a new email and says to check that inbox', async () => {
    const api = mockApi({
      [me]: json(200, apiUser()),
      [config]: json(200, { google_enabled: false }),
      [emailChange]: json(202, {}),
    });
    const { user } = renderApp('/profile');

    await user.click(await screen.findByRole('button', { name: 'Change email' }));
    await user.type(screen.getByLabelText('New email'), 'ada@new.example');
    await user.click(screen.getByRole('button', { name: 'Send confirmation link' }));

    expect(await screen.findByText('Check your new inbox')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveFocus();
    expect(screen.getByText('ada@new.example')).toBeInTheDocument();
    expect(api.callsTo(emailChange)[0]?.body).toEqual({ email: 'ada@new.example' });
  });

  it('refuses the current email without asking the server', async () => {
    const api = mockApi({ [me]: json(200, apiUser()), [config]: json(200, { google_enabled: false }) });
    const { user } = renderApp('/profile');

    await user.click(await screen.findByRole('button', { name: 'Change email' }));
    await user.type(screen.getByLabelText('New email'), 'ADA@example.com');
    await user.click(screen.getByRole('button', { name: 'Send confirmation link' }));

    expect(screen.getByLabelText('New email')).toHaveAccessibleDescription(/That's already your email\./);
    expect(api.callsTo(emailChange)).toHaveLength(0);

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.getByRole('button', { name: 'Change email' })).toHaveFocus();
  });

  it('says when another account has the new email', async () => {
    mockApi({
      [me]: json(200, apiUser()),
      [config]: json(200, { google_enabled: false }),
      [emailChange]: problem(409, { title: 'Conflict' }),
    });
    const { user } = renderApp('/profile');

    await user.click(await screen.findByRole('button', { name: 'Change email' }));
    await user.type(screen.getByLabelText('New email'), 'taken@example.com');
    await user.click(screen.getByRole('button', { name: 'Send confirmation link' }));

    expect(await screen.findByText('Another account already uses that email.')).toBeInTheDocument();
  });

  it('sends the reader to sign in, and back here after, once the session cannot be refreshed', async () => {
    mockApi({
      [me]: problem(401),
      'POST /api/auth/v1/sessions/refresh': problem(401),
      [config]: json(200, { google_enabled: false }),
    });
    const { router } = renderApp('/profile');

    expect(await screen.findByRole('heading', { level: 1, name: 'Sign in' })).toBeInTheDocument();
    expect(screen.getByText('Your session has ended. Sign in again to continue.')).toBeInTheDocument();
    expect(router.state.location.pathname + router.state.location.search).toBe('/sign-in?next=%2Fprofile');
  });

  it('keeps the reader signed in and offers a retry when the server is unreachable', async () => {
    const api = mockApi({ [me]: [offline(), json(200, apiUser())], [config]: json(200, { google_enabled: false }) });
    const { router, user } = renderApp('/profile');

    expect(await screen.findByRole('alert')).toHaveTextContent("Can't reach the server");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('ada@example.com')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/profile');
    expect(api.callsTo('POST /api/auth/v1/sessions/refresh')).toHaveLength(0);
  });
});
