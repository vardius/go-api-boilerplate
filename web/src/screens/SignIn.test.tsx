import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { z } from 'zod';
import { isOwnSignInState } from '../lib/signInStart';
import { json, mockApi, offline, problem } from '../test/api';
import { renderApp } from '../test/render';

const config = 'GET /api/user/v1/config';
const loginLinks = 'POST /api/user/v1/login-links';

describe('Sign in', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('emails a link, says to check the inbox, and links to Mailpit with the reason', async () => {
    const api = mockApi({
      [config]: json(200, { google_enabled: false, mail_preview_url: 'http://localhost:8025' }),
      [loginLinks]: json(202, {}),
    });
    const { user } = renderApp('/sign-in?next=%2Fapps');

    await user.type(screen.getByLabelText('Email'), ' ada@example.com ');
    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Check your inbox' })).toHaveFocus();
    expect(screen.getByText('ada@example.com')).toBeInTheDocument();
    expect(screen.getByText('In local development, emails go to Mailpit')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Open Mailpit/ })).toHaveAttribute('href', 'http://localhost:8025');
    // The state is checked in its own test below.
    expect(api.callsTo(loginLinks)[0]?.body).toMatchObject({ email: 'ada@example.com', next: '/apps' });

    await user.click(screen.getByRole('button', { name: 'Use a different email' }));
    expect(screen.getByLabelText('Email')).toHaveFocus();
  });

  it('says nothing about Mailpit when the server sends real email', async () => {
    mockApi({ [config]: json(200, { google_enabled: false }), [loginLinks]: json(202, {}) });
    const { user } = renderApp('/sign-in');

    await user.type(screen.getByLabelText('Email'), 'ada@example.com');
    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }));

    await screen.findByRole('heading', { name: 'Check your inbox' });
    expect(screen.queryByText(/Mailpit/)).not.toBeInTheDocument();
  });

  it('drops a next that leads to another site', async () => {
    const api = mockApi({ [config]: json(200, { google_enabled: false }), [loginLinks]: json(202, {}) });
    const { user } = renderApp(`/sign-in?next=${encodeURIComponent('https://evil.example/')}`);

    await user.type(screen.getByLabelText('Email'), 'ada@example.com');
    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }));

    await screen.findByRole('heading', { name: 'Check your inbox' });
    expect(api.callsTo(loginLinks)[0]?.body).not.toHaveProperty('next');
  });

  it.each([
    ['denied', 'You said no at Google, so nothing was shared. Sign in with an email link, or try Google again.'],
    ['expired', 'It took too long or was interrupted before it finished. Try Google again, or sign in with an email link.'],
    [
      'unverified',
      "Google hasn't verified this account's email address, and only verified addresses can sign in. Verify it with Google, or sign in with an email link.",
    ],
    [
      'taken',
      'This Google account belongs to another account, or the account with its email address is linked to a different Google account. Sign in with an email link instead.',
    ],
    ['failed', 'Something went wrong while finishing the sign-in. Try again in a moment, or sign in with an email link.'],
  ])('says why Google sign-in ended in %s and what to do', (outcome, text) => {
    mockApi({ [config]: json(200, { google_enabled: true }) });
    renderApp(`/sign-in?google=${outcome}`);

    expect(screen.getByText("Google sign-in didn't finish")).toBeInTheDocument();
    expect(screen.getByText(text)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Email me a sign-in link' })).toBeInTheDocument();
  });

  it('ignores a ?google= value the server never sends', () => {
    mockApi({ [config]: json(200, { google_enabled: true }) });
    renderApp('/sign-in?google=__proto__');

    expect(screen.queryByText("Google sign-in didn't finish")).not.toBeInTheDocument();
  });

  it('keeps Google when the mail preview URL is not a web address', async () => {
    mockApi({ [config]: json(200, { google_enabled: true, mail_preview_url: 'localhost:8025' }) });
    renderApp('/sign-in');

    expect(await screen.findByRole('link', { name: 'Continue with Google' })).toBeInTheDocument();
  });

  it('checks the address before sending anything', async () => {
    const api = mockApi({ [config]: json(200, { google_enabled: false }) });
    const { user } = renderApp('/sign-in');

    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }));

    expect(screen.getByLabelText('Email')).toHaveAccessibleDescription('Enter your email address.');
    expect(screen.getByLabelText('Email')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('Email')).toHaveFocus();
    expect(api.callsTo(loginLinks)).toHaveLength(0);
  });

  it("shows the server's field error under the email", async () => {
    mockApi({
      [config]: json(200, { google_enabled: false }),
      [loginLinks]: problem(400, { errors: { email: 'must be a deliverable address' } }),
    });
    const { user } = renderApp('/sign-in');

    await user.type(screen.getByLabelText('Email'), 'ada@example.invalid');
    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }));

    expect(await screen.findByText('must be a deliverable address')).toBeInTheDocument();
    expect(screen.getByLabelText('Email')).toHaveAccessibleDescription('must be a deliverable address');
    expect(screen.getByLabelText('Email')).toHaveFocus();
  });

  it('offers a retry when the server is unreachable', async () => {
    const api = mockApi({ [config]: json(200, { google_enabled: false }), [loginLinks]: [offline(), json(202, {})] });
    const { user } = renderApp('/sign-in');

    await user.type(screen.getByLabelText('Email'), 'ada@example.com');
    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }));
    expect(await screen.findByRole('alert')).toHaveTextContent("Can't reach the server");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    await screen.findByRole('heading', { name: 'Check your inbox' });
    expect(api.callsTo(loginLinks)).toHaveLength(2);
  });

  it('offers Google when the server has it, keeping the destination', async () => {
    mockApi({ [config]: json(200, { google_enabled: true }) });
    renderApp('/sign-in?next=%2Fapps');

    expect(await screen.findByRole('link', { name: 'Continue with Google' })).toHaveAttribute(
      'href',
      expect.stringMatching(/^\/api\/user\/v1\/google\/start\?next=%2Fapps&state=[A-Za-z0-9_-]{22}$/),
    );
  });
  it('sends a state with each sign-in and remembers it, so its own link goes straight in', async () => {
    const api = mockApi({ [config]: json(200, { google_enabled: true }), [loginLinks]: json(202, {}) });
    const { user } = renderApp('/sign-in');

    await user.type(screen.getByLabelText('Email'), 'ada@example.com');
    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }));
    expect(await screen.findByRole('heading', { level: 1, name: 'Check your inbox' })).toBeInTheDocument();
    const sent = z.object({ state: z.string() }).parse(api.callsTo(loginLinks)[0]?.body).state;
    expect(sent).toMatch(/^[A-Za-z0-9_-]{22}$/);
    expect(isOwnSignInState(sent)).toBe(true);

    await user.click(screen.getByRole('button', { name: 'Use a different email' }));
    const google = screen.getByRole('link', { name: 'Continue with Google' });
    const state = new URL(google.getAttribute('href') ?? '', window.location.origin).searchParams.get('state');
    expect(state).not.toBe(sent);
    // Showing the button remembers nothing; picking it does. jsdom can't follow the link to Google.
    expect(isOwnSignInState(state)).toBe(false);
    google.addEventListener('click', (event) => event.preventDefault());
    await user.click(google);
    expect(isOwnSignInState(state)).toBe(true);
  });
});
