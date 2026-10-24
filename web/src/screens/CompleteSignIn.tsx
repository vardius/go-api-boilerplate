import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';
import { useIntl } from 'react-intl';
import { useNavigate, useSearchParams } from 'react-router';
import { isProblem } from '../api/describe';
import { createSession } from '../api/endpoints';
import { useSignedInUser } from '../api/queries';
import { loadPage } from '../lib/browser';
import { isServerPath, safeNext, signInPath } from '../lib/next';
import { Actions, Button, ButtonAnchor, ButtonLink } from '../ui/Button';
import { ErrorState } from '../ui/ErrorState';
import { Loading } from '../ui/Loading';
import { Card, Page } from '../ui/Page';

/**
 * Where a sign-in link, or a Google sign-in, lands: trades its one-time code
 * for a session, then moves on.
 *
 * Signing in replaces this browser's session, so a link someone else sent
 * could quietly sign the reader in to that person's account (login CSRF).
 * With no session the code is redeemed at once; with one, the reader chooses.
 */
export function CompleteSignIn() {
  const intl = useIntl();
  const [params] = useSearchParams();
  const code = params.get('code');
  const next = safeNext(params.get('next'));
  const destination = next ?? '/profile';
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const signedIn = useSignedInUser({ enabled: code !== null && code !== '' });

  const signIn = useMutation({
    mutationFn: createSession,
    onSuccess: () => {
      // Whatever was cached belongs to whoever was signed in before.
      queryClient.clear();
      if (isServerPath(destination)) loadPage(destination);
      else void navigate(destination, { replace: true });
    },
  });

  // The code works once, so it's posted once, even when React runs effects twice in development.
  const posted = useRef(false);
  // The reader chose to switch accounts, so the pages that follow take focus from the button they pressed.
  const [switching, setSwitching] = useState(false);
  const { mutate } = signIn;
  const noSession = signedIn.isSuccess && signedIn.data === null;
  useEffect(() => {
    if (code === null || code === '' || !noSession || posted.current) return;
    posted.current = true;
    mutate(code);
  }, [code, noSession, mutate]);

  const newLink = (
    <Actions>
      <ButtonLink variant="primary" to={signInPath(next)}>
        {intl.formatMessage({ id: 'completeSignIn.newLink' })}
      </ButtonLink>
    </Actions>
  );

  if (code === null || code === '') {
    return (
      <Page title={intl.formatMessage({ id: 'completeSignIn.incomplete.title' })}>
        <Card>
          <p>{intl.formatMessage({ id: 'completeSignIn.incomplete.body' })}</p>
          {newLink}
        </Card>
      </Page>
    );
  }

  // 401 is the server's answer for a code that is wrong, used or past its 15 minutes.
  if (isProblem(signIn.error, 400, 401, 404, 410)) {
    return (
      <Page title={intl.formatMessage({ id: 'completeSignIn.expired.title' })} focusOnMount={switching}>
        <Card>
          <p>{intl.formatMessage({ id: 'completeSignIn.expired.body' })}</p>
          {newLink}
        </Card>
      </Page>
    );
  }

  if (signIn.isError) {
    return (
      <Page title={intl.formatMessage({ id: 'completeSignIn.failed.title' })} focusOnMount={switching}>
        <ErrorState error={signIn.error} onRetry={() => mutate(code)} />
      </Page>
    );
  }

  const current = signedIn.data;
  if (signIn.isIdle && current !== undefined && current !== null) {
    const stayLabel = intl.formatMessage({ id: 'switchAccount.stay' }, { email: current.email });
    return (
      <Page title={intl.formatMessage({ id: 'switchAccount.title' })}>
        <Card>
          <p>
            {intl.formatMessage(
              { id: 'switchAccount.body' },
              { email: <strong key="email">{current.email}</strong> },
            )}
          </p>
          <Actions>
            <Button
              variant="primary"
              onClick={() => {
                posted.current = true;
                setSwitching(true);
                mutate(code);
              }}
            >
              {intl.formatMessage({ id: 'switchAccount.continue' })}
            </Button>
            {isServerPath(destination) ? (
              <ButtonAnchor href={destination}>{stayLabel}</ButtonAnchor>
            ) : (
              <ButtonLink to={destination}>{stayLabel}</ButtonLink>
            )}
          </Actions>
        </Card>
      </Page>
    );
  }

  // Without knowing whether someone is signed in, the code isn't redeemed.
  if (signIn.isIdle && signedIn.isError) {
    return (
      <Page title={intl.formatMessage({ id: 'completeSignIn.failed.title' })}>
        <ErrorState error={signedIn.error} onRetry={() => void signedIn.refetch()} />
      </Page>
    );
  }

  return (
    <Page title={intl.formatMessage({ id: 'completeSignIn.pending.title' })} focusOnMount={switching}>
      <Loading label={intl.formatMessage({ id: 'completeSignIn.pending.body' })} />
    </Page>
  );
}
