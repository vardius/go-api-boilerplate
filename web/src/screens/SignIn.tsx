import { useMutation } from '@tanstack/react-query';
import { useState, type SubmitEvent } from 'react';
import { useIntl } from 'react-intl';
import { useLocation, useSearchParams } from 'react-router';
import { z } from 'zod';
import { isProblem } from '../api/describe';
import { googleStartURL, requestLoginLink } from '../api/endpoints';
import { useConfig } from '../api/queries';
import type { MessageId } from '../i18n/messages';
import { emailProblem } from '../lib/email';
import { googleFailure } from '../lib/google';
import { safeNext } from '../lib/next';
import { newSignInState, rememberSignInState } from '../lib/signInStart';
import { Actions, Button, ButtonAnchor } from '../ui/Button';
import { ErrorState } from '../ui/ErrorState';
import { TextField } from '../ui/Field';
import { Notice } from '../ui/Notice';
import { Card, Page, Stack } from '../ui/Page';
import { useFocusFirstInvalid } from '../ui/useFocusFirstInvalid';
import { GoogleFailureNotice } from './GoogleFailureNotice';
import { MailPreviewNotice } from './MailPreviewNotice';
import styles from './screens.module.css';

const reasonSchema = z.object({ reason: z.enum(['session-expired', 'signed-out']) });

const reasonMessages: Record<z.infer<typeof reasonSchema>['reason'], MessageId> = {
  'session-expired': 'signIn.sessionExpired',
  'signed-out': 'signIn.signedOut',
};

export function SignIn() {
  const intl = useIntl();
  const [params] = useSearchParams();
  const next = safeNext(params.get('next'));
  const google = googleFailure(params.get('google'));
  const reason = reasonSchema.safeParse(useLocation().state);
  const config = useConfig();
  const [email, setEmail] = useState('');
  const [invalid, setInvalid] = useState<MessageId>();
  const send = useMutation({
    mutationFn: (address: string) => {
      const state = newSignInState();
      rememberSignInState(state);
      return requestLoginLink(address, next, state);
    },
  });
  const { form, submitted } = useFocusFirstInvalid(send.error);
  // Remembered only when the reader picks Google, so a page left open marks nothing.
  const [googleState] = useState(newSignInState);
  // Back from "check your inbox" by choice: the email field is what the reader came back for.
  const [again, setAgain] = useState(false);

  if (send.isSuccess) {
    return (
      <Page title={intl.formatMessage({ id: 'checkInbox.title' })} focusOnMount>
        <Card>
          <p>
            {intl.formatMessage(
              { id: 'checkInbox.body' },
              { email: <strong key="email">{send.variables}</strong> },
            )}
          </p>
          <p className={styles.muted}>{intl.formatMessage({ id: 'checkInbox.spam' })}</p>
          <Actions>
            <Button
              onClick={() => {
                setAgain(true);
                send.reset();
              }}
            >
              {intl.formatMessage({ id: 'checkInbox.differentEmail' })}
            </Button>
          </Actions>
        </Card>
        <MailPreviewNotice />
      </Page>
    );
  }

  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const address = email.trim();
    const problem = emailProblem(address);
    setInvalid(problem);
    submitted();
    if (problem === undefined) send.mutate(address);
  }

  const serverFieldError = isProblem(send.error, 400) ? send.error.fieldErrors.email : undefined;
  const fieldError = invalid === undefined ? serverFieldError : intl.formatMessage({ id: invalid });
  const formError = send.isError && serverFieldError === undefined ? send.error : undefined;

  return (
    <Page title={intl.formatMessage({ id: 'signIn.title' })} lead={intl.formatMessage({ id: 'signIn.lead' })}>
      {reason.success && (
        <Notice tone={reason.data.reason === 'signed-out' ? 'success' : 'info'} live="status">
          <p>{intl.formatMessage({ id: reasonMessages[reason.data.reason] })}</p>
        </Notice>
      )}
      {google !== undefined && <GoogleFailureNotice failure={google} linking={false} />}
      <Card>
        <form ref={form} className={styles.form} noValidate onSubmit={submit}>
          <TextField
            label={intl.formatMessage({ id: 'signIn.email' })}
            type="email"
            name="email"
            autoComplete="email"
            inputMode="email"
            spellCheck={false}
            autoCapitalize="none"
            autoFocus={again}
            required
            value={email}
            error={fieldError}
            onChange={(event) => setEmail(event.target.value)}
          />
          {formError !== undefined && (
            <ErrorState error={formError} onRetry={() => send.mutate(email.trim())} />
          )}
          <Actions>
            <Button type="submit" variant="primary" disabled={send.isPending}>
              {intl.formatMessage({ id: send.isPending ? 'signIn.sending' : 'signIn.submit' })}
            </Button>
          </Actions>
        </form>
        {config.data?.google_enabled === true && (
          <Stack>
            <p className={styles.divider}>{intl.formatMessage({ id: 'signIn.or' })}</p>
            <Actions>
              <ButtonAnchor
                href={googleStartURL({ next, state: googleState })}
                onClick={() => rememberSignInState(googleState)}
              >
                {intl.formatMessage({ id: 'signIn.google' })}
              </ButtonAnchor>
            </Actions>
          </Stack>
        )}
      </Card>
    </Page>
  );
}
