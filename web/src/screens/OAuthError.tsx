import { useIntl } from 'react-intl';
import { useSearchParams } from 'react-router';
import type { MessageId } from '../i18n/messages';
import { Actions, ButtonLink } from '../ui/Button';
import { Card, Page } from '../ui/Page';
import styles from './screens.module.css';

// A Map, so a reason like "constructor" can't reach Object.prototype.
const explanations = new Map<string, MessageId>([
  ['invalid_client', 'oauthError.invalid_client'],
  ['invalid_redirect_uri', 'oauthError.invalid_redirect_uri'],
]);

/** Where the auth service sends a browser whose authorize request it can't safely redirect back to the app. */
export function OAuthError() {
  const intl = useIntl();
  const [params] = useSearchParams();
  const raw = params.get('reason') ?? '';
  // Only an error code is shown, never arbitrary text from the address bar.
  const reason = /^[a-z0-9_.-]{1,64}$/i.test(raw) ? raw : undefined;
  const explanation = (reason === undefined ? undefined : explanations.get(reason)) ?? 'oauthError.generic';

  return (
    <Page title={intl.formatMessage({ id: 'oauthError.title' })}>
      <Card>
        <p>{intl.formatMessage({ id: explanation })}</p>
        <p>{intl.formatMessage({ id: reason === undefined ? 'oauthError.adviceNoCode' : 'oauthError.advice' })}</p>
        {reason !== undefined && (
          <p className={styles.muted}>
            {intl.formatMessage({ id: 'oauthError.code' }, { code: <code key="code">{reason}</code> })}
          </p>
        )}
        <Actions>
          <ButtonLink variant="primary" to="/">
            {intl.formatMessage({ id: 'oauthError.home' })}
          </ButtonLink>
        </Actions>
      </Card>
    </Page>
  );
}
