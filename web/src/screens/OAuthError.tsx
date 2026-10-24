import { useIntl } from 'react-intl';
import { useSearchParams } from 'react-router';
import type { MessageId } from '../i18n/messages';
import { Actions, ButtonLink } from '../ui/Button';
import { Card, Page } from '../ui/Page';
import styles from './screens.module.css';

/** What a reason means to the reader, and what they can do about it. */
interface Explanation {
  explanation: MessageId;
  advice: MessageId;
}

// The app is at fault: its developer has to fix the request.
const appFault = (explanation: MessageId): Explanation => ({ explanation, advice: 'oauthError.advice' });
// Nothing is wrong with the app: starting again from it works.
const startAgain = (explanation: MessageId): Explanation => ({ explanation, advice: 'oauthError.adviceStartAgain' });

// Exactly the reasons the auth service sends (app.Reasons in internal/auth/app;
// a test in internal/auth/httpapi checks this list). A Map, so a reason like
// "constructor" can't reach Object.prototype.
const reasons = new Map<string, Explanation>([
  ['invalid_client', appFault('oauthError.invalid_client')],
  ['invalid_redirect_uri', appFault('oauthError.invalid_redirect_uri')],
  ['invalid_request', appFault('oauthError.invalid_request')],
  ['pkce_required', appFault('oauthError.pkce_required')],
  ['unsupported_response_type', appFault('oauthError.unsupported_response_type')],
  ['invalid_scope', appFault('oauthError.invalid_scope')],
  ['login_required', startAgain('oauthError.login_required')],
  ['request_expired', startAgain('oauthError.request_expired')],
]);

/** Where the auth service sends a browser whose authorize request it refused. */
export function OAuthError() {
  const intl = useIntl();
  const [params] = useSearchParams();
  // Only a reason from the list is shown, never other text from the address bar.
  const reason = params.get('reason') ?? '';
  const known = reasons.get(reason);

  return (
    <Page title={intl.formatMessage({ id: 'oauthError.title' })}>
      <Card>
        <p>{intl.formatMessage({ id: known?.explanation ?? 'oauthError.generic' })}</p>
        <p>{intl.formatMessage({ id: known?.advice ?? 'oauthError.adviceNoCode' })}</p>
        {known !== undefined && (
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
