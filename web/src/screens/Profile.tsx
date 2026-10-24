import { useMutation } from '@tanstack/react-query';
import { useEffect, useId, useRef, useState, type SubmitEvent } from 'react';
import { useIntl } from 'react-intl';
import { useSearchParams } from 'react-router';
import { isProblem } from '../api/describe';
import { googleStartURL, requestEmailChange } from '../api/endpoints';
import { useConfig, useMe } from '../api/queries';
import type { User } from '../api/schemas';
import type { MessageId } from '../i18n/messages';
import { emailProblem } from '../lib/email';
import { googleFailure } from '../lib/google';
import { Actions, Button, ButtonAnchor } from '../ui/Button';
import { ErrorState } from '../ui/ErrorState';
import { TextField } from '../ui/Field';
import { Loading } from '../ui/Loading';
import { Notice } from '../ui/Notice';
import { Muted, Page, Section } from '../ui/Page';
import { useFocusFirstInvalid } from '../ui/useFocusFirstInvalid';
import { GoogleFailureNotice } from './GoogleFailureNotice';
import { MailPreviewNotice } from './MailPreviewNotice';
import styles from './screens.module.css';

export function Profile() {
  const intl = useIntl();
  const [params] = useSearchParams();
  const me = useMe();
  const google = googleFailure(params.get('google'));

  return (
    <Page title={intl.formatMessage({ id: 'profile.title' })}>
      {params.get('google') === 'linked' && (
        <Notice tone="success" live="status">
          <p>{intl.formatMessage({ id: 'profile.google.linkedNotice' })}</p>
        </Notice>
      )}
      {google !== undefined && <GoogleFailureNotice failure={google} linking />}
      {me.isError && <ErrorState error={me.error} onRetry={() => void me.refetch()} />}
      {me.data === undefined ? (
        !me.isError && <Loading />
      ) : (
        <>
          <EmailSection user={me.data} />
          <GoogleSection linked={me.data.google_linked} />
          <Muted>
            {intl.formatMessage(
              { id: 'profile.memberSince' },
              { date: intl.formatDate(me.data.created_at, { dateStyle: 'long' }) },
            )}
          </Muted>
        </>
      )}
    </Page>
  );
}

function EmailSection({ user }: { user: User }) {
  const intl = useIntl();
  const [editing, setEditing] = useState(false);
  const [email, setEmail] = useState('');
  const [invalid, setInvalid] = useState<MessageId>();
  const change = useMutation({ mutationFn: requestEmailChange });
  const { form, submitted } = useFocusFirstInvalid(change.error);

  // Cancelling removes the focused form; land back on the button that opened it.
  // (Sending lands on the sent notice instead, which takes focus itself.)
  const changeButton = useRef<HTMLButtonElement>(null);
  const focusChangeButton = useRef(false);
  useEffect(() => {
    if (editing || !focusChangeButton.current) return;
    focusChangeButton.current = false;
    changeButton.current?.focus();
  }, [editing]);

  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const address = email.trim();
    const problem =
      emailProblem(address) ??
      (address.toLowerCase() === user.email.toLowerCase() ? 'profile.email.same' : undefined);
    setInvalid(problem);
    submitted();
    if (problem !== undefined) return;
    change.mutate(address, {
      onSuccess: () => {
        setEditing(false);
        setEmail('');
      },
    });
  }

  const serverFieldError = isProblem(change.error, 400) ? change.error.fieldErrors.email : undefined;
  const fieldError = invalid === undefined ? serverFieldError : intl.formatMessage({ id: invalid });
  const formError = change.isError && fieldError === undefined ? change.error : undefined;

  return (
    <Section title={intl.formatMessage({ id: 'profile.email.heading' })}>
      <p className={styles.value}>{user.email}</p>
      {change.isSuccess && !editing && (
        <>
          <Notice
            tone="success"
            live="status"
            focusOnMount
            title={intl.formatMessage({ id: 'profile.email.sent.title' })}
          >
            <p>
              {intl.formatMessage(
                { id: 'profile.email.sent.body' },
                { email: <strong key="email">{change.variables}</strong>, current: user.email },
              )}
            </p>
          </Notice>
          <MailPreviewNotice />
        </>
      )}
      {editing ? (
        <form ref={form} className={styles.form} noValidate onSubmit={submit}>
          <TextField
            label={intl.formatMessage({ id: 'profile.email.new' })}
            hint={intl.formatMessage({ id: 'profile.email.newHint' })}
            type="email"
            name="email"
            autoComplete="email"
            inputMode="email"
            spellCheck={false}
            autoCapitalize="none"
            // The field appears because the reader asked for it, so focus goes there.
            autoFocus
            value={email}
            error={fieldError}
            onChange={(event) => setEmail(event.target.value)}
          />
          {formError !== undefined && <ErrorState error={formError} onRetry={() => change.mutate(email.trim())} />}
          <Actions>
            <Button type="submit" variant="primary" disabled={change.isPending}>
              {intl.formatMessage({ id: change.isPending ? 'profile.email.sending' : 'profile.email.submit' })}
            </Button>
            <Button
              onClick={() => {
                focusChangeButton.current = true;
                setEditing(false);
                setInvalid(undefined);
                change.reset();
              }}
              disabled={change.isPending}
            >
              {intl.formatMessage({ id: 'common.cancel' })}
            </Button>
          </Actions>
        </form>
      ) : (
        <Actions>
          <Button ref={changeButton} variant="primary" onClick={() => setEditing(true)}>
            {intl.formatMessage({ id: 'profile.email.change' })}
          </Button>
        </Actions>
      )}
    </Section>
  );
}

function GoogleSection({ linked }: { linked: boolean }) {
  const intl = useIntl();
  const config = useConfig();
  const reasonId = useId();

  let body;
  if (linked) {
    body = <p>{intl.formatMessage({ id: 'profile.google.linked' })}</p>;
  } else if (config.data === undefined) {
    body = config.isError ? (
      <ErrorState
        error={config.error}
        title={intl.formatMessage({ id: 'profile.google.configError' })}
        onRetry={() => void config.refetch()}
      />
    ) : (
      <Loading />
    );
  } else if (config.data.google_enabled) {
    body = (
      <>
        <p>{intl.formatMessage({ id: 'profile.google.unlinked' })}</p>
        <Actions>
          <ButtonAnchor href={googleStartURL({ link: true })}>
            {intl.formatMessage({ id: 'profile.google.link' })}
          </ButtonAnchor>
        </Actions>
      </>
    );
  } else {
    body = (
      <>
        <Actions>
          {/* aria-disabled rather than disabled: the control stays focusable, so its reason is heard. */}
          <Button aria-disabled="true" aria-describedby={reasonId}>
            {intl.formatMessage({ id: 'profile.google.link' })}
          </Button>
        </Actions>
        <p id={reasonId} className={styles.muted}>
          {intl.formatMessage({ id: 'profile.google.disabledReason' })}
        </p>
      </>
    );
  }

  return <Section title={intl.formatMessage({ id: 'profile.google.heading' })}>{body}</Section>;
}
