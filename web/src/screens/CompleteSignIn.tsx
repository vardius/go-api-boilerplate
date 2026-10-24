import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { useIntl } from 'react-intl';
import { useNavigate, useSearchParams } from 'react-router';
import { isProblem } from '../api/describe';
import { createSession } from '../api/endpoints';
import { loadPage } from '../lib/browser';
import { isServerPath, safeNext, signInPath } from '../lib/next';
import { Actions, ButtonLink } from '../ui/Button';
import { ErrorState } from '../ui/ErrorState';
import { Loading } from '../ui/Loading';
import { Card, Page } from '../ui/Page';

/** Where a sign-in link lands: trades its one-time code for a session, then moves on. */
export function CompleteSignIn() {
  const intl = useIntl();
  const [params] = useSearchParams();
  const code = params.get('code');
  const next = safeNext(params.get('next'));
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const signIn = useMutation({
    mutationFn: createSession,
    onSuccess: () => {
      // Whatever was cached belongs to whoever was signed in before.
      queryClient.clear();
      const destination = next ?? '/profile';
      if (isServerPath(destination)) loadPage(destination);
      else void navigate(destination, { replace: true });
    },
  });

  // The code works once, so it's posted once, even when React runs effects twice in development.
  const posted = useRef(false);
  const { mutate } = signIn;
  useEffect(() => {
    if (code === null || code === '' || posted.current) return;
    posted.current = true;
    mutate(code);
  }, [code, mutate]);

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
      <Page title={intl.formatMessage({ id: 'completeSignIn.expired.title' })}>
        <Card>
          <p>{intl.formatMessage({ id: 'completeSignIn.expired.body' })}</p>
          {newLink}
        </Card>
      </Page>
    );
  }

  if (signIn.isError) {
    return (
      <Page title={intl.formatMessage({ id: 'completeSignIn.failed.title' })}>
        <ErrorState error={signIn.error} onRetry={() => mutate(code)} />
      </Page>
    );
  }

  return (
    <Page title={intl.formatMessage({ id: 'completeSignIn.pending.title' })}>
      <Loading label={intl.formatMessage({ id: 'completeSignIn.pending.body' })} />
    </Page>
  );
}
