import { useRef, useState } from 'react';
import { useIntl } from 'react-intl';
import { useSearchParams } from 'react-router';
import { isProblem } from '../api/describe';
import { authorizeFormAction } from '../api/endpoints';
import { useAuthorizationRequest } from '../api/queries';
import type { AuthorizationRequest } from '../api/schemas';
import { scopeDescriptionId } from '../i18n/scopes';
import { Actions, Button, ButtonLink } from '../ui/Button';
import { ErrorState } from '../ui/ErrorState';
import { Loading } from '../ui/Loading';
import { Card, Page } from '../ui/Page';
import styles from './screens.module.css';

/** Asks the signed-in reader whether a third-party app may act for them. */
export function Consent() {
  const intl = useIntl();
  const [params] = useSearchParams();
  const id = params.get('request');
  const pageTitle = intl.formatMessage({ id: 'consent.pageTitle' });

  if (id === null || id === '') {
    return (
      <Page title={intl.formatMessage({ id: 'consent.incomplete.title' })}>
        <Card>
          <p>{intl.formatMessage({ id: 'consent.incomplete.body' })}</p>
          <HomeLink />
        </Card>
      </Page>
    );
  }
  return <ConsentRequest id={id} pageTitle={pageTitle} />;
}

function ConsentRequest({ id, pageTitle }: { id: string; pageTitle: string }) {
  const intl = useIntl();
  const request = useAuthorizationRequest(id);

  if (request.data !== undefined) return <ConsentForm id={id} request={request.data} pageTitle={pageTitle} />;
  if (!request.isError) {
    return (
      <Page title={pageTitle}>
        <Loading />
      </Page>
    );
  }
  if (isProblem(request.error, 404, 410)) {
    return (
      <Page title={intl.formatMessage({ id: 'consent.expired.title' })}>
        <Card>
          <p>{intl.formatMessage({ id: 'consent.expired.body' })}</p>
          <HomeLink />
        </Card>
      </Page>
    );
  }
  return (
    <Page title={intl.formatMessage({ id: 'consent.failed.title' })}>
      <ErrorState error={request.error} onRetry={() => void request.refetch()} />
    </Page>
  );
}

/** The way out when there is nothing left to decide; the app that sent the reader here is unknown. */
function HomeLink() {
  const intl = useIntl();
  return (
    <Actions>
      <ButtonLink variant="primary" to="/">
        {intl.formatMessage({ id: 'consent.home' })}
      </ButtonLink>
    </Actions>
  );
}

function ConsentForm({ id, request, pageTitle }: { id: string; request: AuthorizationRequest; pageTitle: string }) {
  const intl = useIntl();
  const app = request.client.name;
  const submitted = useRef(false);
  const [redirecting, setRedirecting] = useState(false);

  return (
    <Page title={intl.formatMessage({ id: 'consent.title' }, { app })} documentTitle={pageTitle}>
      <Card>
        {request.scopes.length === 0 ? (
          <p>{intl.formatMessage({ id: 'consent.noScopes' }, { app })}</p>
        ) : (
          <>
            <p>{intl.formatMessage({ id: 'consent.asks' }, { app })}</p>
            <ul className={styles.scopes}>
              {request.scopes.map((scope) => {
                const known = scopeDescriptionId(scope.name);
                return (
                  <li key={scope.name} className={styles.scope}>
                    <span>{known === undefined ? scope.description : intl.formatMessage({ id: known })}</span>
                    <span className={styles.scopeName}>{scope.name}</span>
                  </li>
                );
              })}
            </ul>
          </>
        )}
        <p>
          {intl.formatMessage(
            { id: 'consent.redirect' },
            { host: <strong key="host">{request.redirect_host}</strong> },
          )}
        </p>
        <p className={styles.muted}>{intl.formatMessage({ id: 'consent.trust' })}</p>
        {/*
          A real form post, not fetch: the answer is a redirect to the app's own site,
          which the browser must follow as a navigation.
        */}
        <form
          method="post"
          action={authorizeFormAction}
          onSubmit={(event) => {
            // The request is single use; a second click would only reach an error.
            if (submitted.current) {
              event.preventDefault();
              return;
            }
            submitted.current = true;
            setRedirecting(true);
          }}
        >
          <input type="hidden" name="request" value={id} />
          <Actions>
            <Button type="submit" variant="primary" name="decision" value="allow">
              {intl.formatMessage({ id: 'consent.allow' })}
            </Button>
            <Button type="submit" name="decision" value="deny">
              {intl.formatMessage({ id: 'consent.deny' })}
            </Button>
          </Actions>
        </form>
        {redirecting && (
          <Loading label={intl.formatMessage({ id: 'consent.redirecting' }, { host: request.redirect_host })} />
        )}
      </Card>
    </Page>
  );
}
