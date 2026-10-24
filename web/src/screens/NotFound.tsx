import { useIntl } from 'react-intl';
import { Actions, ButtonLink } from '../ui/Button';
import { Card, Page } from '../ui/Page';

export function NotFound() {
  const intl = useIntl();
  return (
    <Page title={intl.formatMessage({ id: 'notFound.title' })}>
      <Card>
        <p>{intl.formatMessage({ id: 'notFound.body' })}</p>
        <Actions>
          <ButtonLink variant="primary" to="/">
            {intl.formatMessage({ id: 'notFound.home' })}
          </ButtonLink>
        </Actions>
      </Card>
    </Page>
  );
}
