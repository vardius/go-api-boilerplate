import { useIntl } from 'react-intl';
import { Actions, Button } from '../ui/Button';
import { Card, Page } from '../ui/Page';

/** Shown when a screen throws while rendering: a bug, not an API failure. */
export function RouteError() {
  const intl = useIntl();
  return (
    <Page title={intl.formatMessage({ id: 'error.page.title' })}>
      <Card>
        <p>{intl.formatMessage({ id: 'error.page.body' })}</p>
        <Actions>
          <Button variant="primary" onClick={() => window.location.reload()}>
            {intl.formatMessage({ id: 'error.page.reload' })}
          </Button>
        </Actions>
      </Card>
    </Page>
  );
}
