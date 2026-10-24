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

  it.each([
    ['invalid_client', "The app that sent you here isn't registered here, or it was deleted.", 'developer'],
    ['invalid_redirect_uri', "The app that sent you here asked to return to an address it didn't register.", 'developer'],
    ['invalid_request', 'The app that sent you here made a request that is incomplete or malformed.', 'developer'],
    ['pkce_required', "The app that sent you here didn't secure its request the way we require (PKCE with S256).", 'developer'],
    ['unsupported_response_type', "The app that sent you here asked for a kind of sign-in we don't offer.", 'developer'],
    ['invalid_scope', "The app that sent you here asked for access it isn't registered for, or that doesn't exist.", 'developer'],
    ['login_required', "You weren't signed in when you answered, so nothing was decided.", 'start again'],
    ['request_expired', 'This request is no longer waiting for an answer: it expired or was already answered.', 'start again'],
  ])('explains %s', (reason, explanation, advice) => {
    renderApp(`/oauth/error?reason=${reason}`);

    expect(screen.getByText(explanation)).toBeInTheDocument();
    expect(screen.getByText(new RegExp(advice))).toBeInTheDocument();
    expect(screen.getByText(reason)).toBeInTheDocument();
  });

  it('falls back to a general explanation and hides a reason that is not a code', () => {
    renderApp(`/oauth/error?reason=${encodeURIComponent('<b>pay here</b>')}`);

    expect(screen.getByText("The app that sent you here made a request we can't accept.")).toBeInTheDocument();
    expect(screen.queryByText(/pay here/)).not.toBeInTheDocument();
  });

  it('shows no code the auth service does not send', () => {
    renderApp('/oauth/error?reason=call_support_now');

    expect(screen.getByText("The app that sent you here made a request we can't accept.")).toBeInTheDocument();
    expect(screen.getByText('Nothing was shared with the app. Let its developer know.')).toBeInTheDocument();
    expect(screen.queryByText(/call_support_now/)).not.toBeInTheDocument();
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
