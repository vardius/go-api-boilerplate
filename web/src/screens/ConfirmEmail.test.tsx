import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { json, mockApi, problem } from '../test/api';
import { apiUser, renderApp } from '../test/render';

const confirm = 'POST /api/user/v1/me/email-change/confirm';

describe('Confirm email', () => {
  it('confirms the new address once and shows it', async () => {
    const api = mockApi({ [confirm]: json(200, apiUser({ email: 'ada@new.example' })) });
    renderApp('/email/confirm?code=c1');

    expect(await screen.findByRole('heading', { name: 'Email changed' })).toBeInTheDocument();
    expect(screen.getByText('You now sign in with ada@new.example.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to your profile' })).toHaveAttribute('href', '/profile');
    expect(api.callsTo(confirm)).toEqual([expect.objectContaining({ body: { code: 'c1' } })]);
  });

  it('explains an expired or used link', async () => {
    mockApi({ [confirm]: problem(400, { detail: 'The code is invalid or expired.' }) });
    renderApp('/email/confirm?code=old');

    expect(await screen.findByRole('heading', { name: 'This confirmation link has expired' })).toBeInTheDocument();
  });

  it('says when the address was taken in the meantime', async () => {
    mockApi({ [confirm]: problem(409) });
    renderApp('/email/confirm?code=c1');

    expect(await screen.findByRole('heading', { name: 'That email is taken' })).toBeInTheDocument();
  });

  it('says when the link has no code', () => {
    const api = mockApi();
    renderApp('/email/confirm');

    expect(screen.getByRole('heading', { name: 'This confirmation link is incomplete' })).toBeInTheDocument();
    expect(api.calls).toHaveLength(0);
  });
});
