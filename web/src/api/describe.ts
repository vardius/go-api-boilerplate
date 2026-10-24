import type { MessageId } from '../i18n/messages';
import { NetworkError, ProblemError, SessionExpiredError, UnexpectedResponseError } from './client';

export interface ErrorDescription {
  message: MessageId;
  /** Whether asking again could help. A network error always can: it never means signed out. */
  retryable: boolean;
  /** The server's own explanation, when it gave one for a request it refused. */
  detail?: string | undefined;
  /** For a server fault, the ID to quote when reporting it. */
  requestId?: string | undefined;
}

export function describeError(error: unknown): ErrorDescription {
  if (error instanceof NetworkError) return { message: 'error.network', retryable: true };
  if (error instanceof SessionExpiredError) return { message: 'error.sessionEnded', retryable: false };
  if (error instanceof UnexpectedResponseError) return { message: 'error.unexpected', retryable: true };
  if (!(error instanceof ProblemError)) return { message: 'error.unknown', retryable: true };
  if (error.status >= 500) return { message: 'error.server', retryable: true, requestId: error.requestId };
  switch (error.status) {
    case 429:
      return { message: 'error.rateLimited', retryable: true };
    case 403:
      return { message: 'error.forbidden', retryable: false };
    case 404:
      return { message: 'error.notFound', retryable: false };
    default:
      return { message: 'error.request', retryable: false, detail: error.detail };
  }
}

export function isProblem(error: unknown, ...statuses: number[]): error is ProblemError {
  return error instanceof ProblemError && statuses.includes(error.status);
}
