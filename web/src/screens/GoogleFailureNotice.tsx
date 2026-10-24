import { useIntl } from 'react-intl';
import type { MessageId } from '../i18n/messages';
import type { GoogleFailure } from '../lib/google';
import { Notice } from '../ui/Notice';

const signIn: Record<GoogleFailure, MessageId> = {
  denied: 'signIn.googleFailed.denied',
  expired: 'signIn.googleFailed.expired',
  unverified: 'signIn.googleFailed.unverified',
  taken: 'signIn.googleFailed.taken',
  failed: 'signIn.googleFailed.failed',
};

const linking: Record<GoogleFailure, MessageId> = {
  denied: 'profile.google.failed.denied',
  expired: 'profile.google.failed.expired',
  unverified: 'profile.google.failed.unverified',
  taken: 'profile.google.failed.taken',
  failed: 'profile.google.failed.failed',
};

/** Why a Google sign-in (on /sign-in) or link (on /profile) didn't finish, and what to do next. */
export function GoogleFailureNotice({ failure, linking: isLinking }: { failure: GoogleFailure; linking: boolean }) {
  const intl = useIntl();
  return (
    <Notice
      tone="warning"
      title={intl.formatMessage({ id: isLinking ? 'profile.google.failed.title' : 'signIn.googleFailed.title' })}
    >
      <p>{intl.formatMessage({ id: (isLinking ? linking : signIn)[failure] })}</p>
    </Notice>
  );
}
