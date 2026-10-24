import { useIntl } from 'react-intl';
import { useConfig } from '../api/queries';
import { Notice } from '../ui/Notice';

/** In local development the server sends mail to Mailpit; say so and link there. Renders nothing elsewhere. */
export function MailPreviewNotice() {
  const intl = useIntl();
  const url = useConfig().data?.mail_preview_url;
  if (url === undefined) return null;
  return (
    <Notice tone="info" title={intl.formatMessage({ id: 'mailPreview.title' })}>
      <p>{intl.formatMessage({ id: 'mailPreview.body' })}</p>
      <p>
        <a href={url} target="_blank" rel="noopener noreferrer">
          {intl.formatMessage({ id: 'mailPreview.link' })}
          <span className="visually-hidden"> {intl.formatMessage({ id: 'common.newTab' })}</span>
        </a>
      </p>
    </Notice>
  );
}
