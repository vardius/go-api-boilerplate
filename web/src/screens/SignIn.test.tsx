import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { json, mockApi, offline, problem } from '../test/api';
import { renderApp } from '../test/render';

const config = 'GET /api/user/v1/config';
const loginLinks = 'POST /api/user/v1/login-links';

describe('Sign in', () => {
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
    expect(api.callsTo(loginLinks)[0]?.body).toEqual({ email: 'ada@example.com', next: '/apps' });

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
    expect(api.callsTo(loginLinks)[0]?.body).toEqual({ email: 'ada@example.com' });
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
      '/api/user/v1/google/start?next=%2Fapps',
    );
  });
});
