import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { json, mockApi, problem } from '../test/api';
import { apiUser, renderApp } from '../test/render';

const confirm = 'POST /api/user/v1/me/email-change/confirm';

describe('Confirm email', () => {
  it('confirms the new address once and shows it', async () => {
    const api = mockApi({ [confirm]: json(200, apiUser({ email: 'ada@new.example' })) });
    renderApp('/profile/confirm-email?code=c1');

    expect(await screen.findByRole('heading', { name: 'Email changed' })).toBeInTheDocument();
    expect(screen.getByText('You now sign in with ada@new.example.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to your profile' })).toHaveAttribute('href', '/profile');
    expect(api.callsTo(confirm)).toEqual([expect.objectContaining({ body: { code: 'c1' } })]);
  });

  it('explains a bad code as an expired or invalid link, and keeps the reader signed in', async () => {
    const api = mockApi({
      [confirm]: problem(400, { title: 'Bad Request', errors: { code: 'is unknown, used or expired' } }),
    });
    renderApp('/profile/confirm-email?code=old');

    expect(await screen.findByRole('heading', { name: 'This confirmation link is expired or invalid' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to your profile' })).toHaveAttribute('href', '/profile');
    expect(screen.getByRole('navigation')).toBeInTheDocument();
    expect(api.callsTo(confirm)).toHaveLength(1);
    expect(api.callsTo('POST /api/auth/v1/sessions/refresh')).toHaveLength(0);
    expect(api.callsTo('DELETE /api/auth/v1/sessions/current')).toHaveLength(0);
  });

  it('says when the address was taken in the meantime', async () => {
    mockApi({ [confirm]: problem(409) });
    renderApp('/profile/confirm-email?code=c1');

    expect(await screen.findByRole('heading', { name: 'That email is taken' })).toBeInTheDocument();
  });

  it('says when the link has no code', () => {
    const api = mockApi();
    renderApp('/profile/confirm-email');

    expect(screen.getByRole('heading', { name: 'This confirmation link is incomplete' })).toBeInTheDocument();
    expect(api.calls).toHaveLength(0);
  });
});
