import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { json, mockApi, problem } from '../test/api';
import { renderApp } from '../test/render';

const details = 'GET /api/auth/v1/authorization-requests/req-1';
const request = {
  client: { id: 'client-1', name: 'Acme Notes' },
  scopes: [
    { name: 'user:read', description: 'Read your profile' },
    { name: 'notes:sync', description: 'Sync your notes' },
  ],
  redirect_host: 'notes.example',
};

describe('Consent', () => {
  it('names the app, what it asks for and where the reader goes next, in a real form post', async () => {
    mockApi({ [details]: json(200, request) });
    const { container } = renderApp('/consent?request=req-1');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Allow Acme Notes to use your account?' }),
    ).toBeInTheDocument();
    // Known scopes are described in the reader's language; others in the server's words.
    expect(screen.getByText('Read your email address and whether Google is linked')).toBeInTheDocument();
    expect(screen.getByText('Sync your notes')).toBeInTheDocument();
    expect(screen.getByText('notes.example')).toBeInTheDocument();

    const form = container.querySelector('form');
    expect(form).toHaveAttribute('method', 'post');
    expect(form).toHaveAttribute('action', '/api/auth/oauth/authorize');
    expect(form?.querySelector('input[type="hidden"][name="request"]')).toHaveValue('req-1');
    expect(screen.getByRole('button', { name: 'Allow' })).toHaveAttribute('value', 'allow');
    expect(screen.getByRole('button', { name: 'Allow' })).toHaveAttribute('name', 'decision');
    expect(screen.getByRole('button', { name: 'Deny' })).toHaveAttribute('value', 'deny');
  });

  it('explains an expired request', async () => {
    mockApi({ [details]: problem(404) });
    renderApp('/consent?request=req-1');

    expect(await screen.findByRole('heading', { name: 'This request has expired' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to your account' })).toHaveAttribute('href', '/');
  });

  it("says when the request couldn't load, with the ID to quote, and loads it on retry", async () => {
    mockApi({ [details]: [problem(503, { request_id: 'req-42' }), json(200, request)] });
    const { user } = renderApp('/consent?request=req-1');

    expect(await screen.findByRole('heading', { name: "We couldn't load this request" })).toBeInTheDocument();
    expect(screen.getByText('req-42')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Allow Acme Notes to use your account?' }),
    ).toBeInTheDocument();
  });

  it('sends one answer, however often it is clicked, and says where the reader is going', async () => {
    mockApi({ [details]: json(200, request) });
    const { container } = renderApp('/consent?request=req-1');
    await screen.findByRole('button', { name: 'Allow' });
    const form = container.querySelector('form');
    if (form === null) throw new Error('The consent form is missing.');

    // fireEvent's answer is false once the submission was cancelled.
    expect(fireEvent.submit(form)).toBe(true);
    expect(await screen.findByRole('status')).toHaveTextContent('Taking you back to notes.example…');
    expect(fireEvent.submit(form)).toBe(false);
  });

  it('says when the link has no request', () => {
    const api = mockApi();
    renderApp('/consent');

    expect(screen.getByRole('heading', { name: 'This request is incomplete' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to your account' })).toHaveAttribute('href', '/');
    expect(api.calls).toHaveLength(0);
  });
});
