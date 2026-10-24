import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { json, mockApi } from '../test/api';
import { renderApp } from '../test/render';

describe('OAuth error', () => {
  it('explains a known reason and shows its code', () => {
    renderApp('/oauth/error?reason=invalid_redirect_uri');

    expect(screen.getByRole('heading', { name: "This app can't sign you in" })).toBeInTheDocument();
    expect(
      screen.getByText("The app that sent you here asked to return to an address it didn't register."),
    ).toBeInTheDocument();
    expect(screen.getByText('invalid_redirect_uri')).toBeInTheDocument();
  });

  it('falls back to a general explanation and hides a reason that is not a code', () => {
    renderApp(`/oauth/error?reason=${encodeURIComponent('<b>pay here</b>')}`);

    expect(screen.getByText("The app that sent you here made a request we can't accept.")).toBeInTheDocument();
    expect(screen.queryByText(/pay here/)).not.toBeInTheDocument();
  });
});

describe('Not found', () => {
  it('answers any unknown path', () => {
    renderApp('/no/such/page');

    expect(screen.getByRole('heading', { name: 'Page not found' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to your account' })).toHaveAttribute('href', '/');
  });
});

describe('Language', () => {
  it('switches the whole app to Polish', async () => {
    mockApi({ 'GET /api/user/v1/config': json(200, { google_enabled: false }) });
    const { user } = renderApp('/sign-in');

    await user.selectOptions(screen.getByLabelText('Language'), 'pl');

    expect(screen.getByRole('heading', { level: 1, name: 'Zaloguj się' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Wyślij mi link do logowania' })).toBeInTheDocument();
    expect(document.documentElement.lang).toBe('pl');
  });
});
