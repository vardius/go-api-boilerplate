import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { useIntl } from 'react-intl';
import { useSearchParams } from 'react-router';
import { isProblem } from '../api/describe';
import { confirmEmailChange } from '../api/endpoints';
import { queryKeys } from '../api/queries';
import type { MessageId } from '../i18n/messages';
import { Actions, ButtonLink } from '../ui/Button';
import { ErrorState } from '../ui/ErrorState';
import { Loading } from '../ui/Loading';
import { Card, Page } from '../ui/Page';

/** Where an email-change link lands: confirms the new address with its one-time code. */
export function ConfirmEmail() {
  const intl = useIntl();
  const [params] = useSearchParams();
  const code = params.get('code');
  const queryClient = useQueryClient();

  const confirm = useMutation({
    mutationFn: confirmEmailChange,
    onSuccess: (user) => queryClient.setQueryData(queryKeys.me, user),
  });

  // The code works once, so it's posted once, even when React runs effects twice in development.
  const posted = useRef(false);
  const { mutate } = confirm;
  useEffect(() => {
    if (code === null || code === '' || posted.current) return;
    posted.current = true;
    mutate(code);
  }, [code, mutate]);

  const toProfile = (
    <Actions>
      <ButtonLink variant="primary" to="/profile">
        {intl.formatMessage({ id: 'confirmEmail.toProfile' })}
      </ButtonLink>
    </Actions>
  );

  function outcome(title: MessageId, body: string) {
    return (
      <Page title={intl.formatMessage({ id: title })}>
        <Card>
          <p>{body}</p>
          {toProfile}
        </Card>
      </Page>
    );
  }

  if (code === null || code === '') {
    return outcome('confirmEmail.incomplete.title', intl.formatMessage({ id: 'confirmEmail.incomplete.body' }));
  }
  if (confirm.isSuccess) {
    return outcome(
      'confirmEmail.done.title',
      intl.formatMessage({ id: 'confirmEmail.done.body' }, { email: confirm.data.email }),
    );
  }
  if (isProblem(confirm.error, 409)) {
    return outcome('confirmEmail.taken.title', intl.formatMessage({ id: 'confirmEmail.taken.body' }));
  }
  if (isProblem(confirm.error, 400, 404, 410)) {
    return outcome('confirmEmail.expired.title', intl.formatMessage({ id: 'confirmEmail.expired.body' }));
  }
  if (confirm.isError) {
    return (
      <Page title={intl.formatMessage({ id: 'confirmEmail.failed.title' })}>
        <ErrorState error={confirm.error} onRetry={() => mutate(code)} />
      </Page>
    );
  }
  return (
    <Page title={intl.formatMessage({ id: 'confirmEmail.pending.title' })}>
      <Loading label={intl.formatMessage({ id: 'confirmEmail.pending.body' })} />
    </Page>
  );
}
