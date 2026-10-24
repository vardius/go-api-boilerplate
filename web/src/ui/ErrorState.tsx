import { useIntl } from 'react-intl';
import { SessionExpiredError } from '../api/client';
import { describeError } from '../api/describe';
import { Button } from './Button';
import { Loading } from './Loading';
import { Notice } from './Notice';

interface ErrorStateProps {
  error: unknown;
  /** Shown only for errors where asking again could help. */
  onRetry?: () => void;
  title?: string;
}

/** Says what failed and, when it could help, offers to try again. */
export function ErrorState({ error, onRetry, title }: ErrorStateProps) {
  const intl = useIntl();
  // The app is already on its way to sign-in; say so instead of showing a failure.
  if (error instanceof SessionExpiredError) return <Loading label={intl.formatMessage({ id: 'error.sessionEnded' })} />;

  const described = describeError(error);
  const message = intl.formatMessage({ id: described.message });
  return (
    <Notice tone="danger" live="alert" title={title ?? message}>
      {title !== undefined && <p>{message}</p>}
      {described.detail !== undefined && <p>{described.detail}</p>}
      {described.requestId !== undefined && (
        <p>{intl.formatMessage({ id: 'error.reference' }, { id: <code key="id">{described.requestId}</code> })}</p>
      )}
      {described.retryable && onRetry !== undefined && (
        <div>
          <Button onClick={onRetry}>{intl.formatMessage({ id: 'common.tryAgain' })}</Button>
        </div>
      )}
    </Notice>
  );
}
