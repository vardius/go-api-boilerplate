import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query';
import type { DataRouter } from 'react-router';
import { NetworkError, ProblemError, SessionExpiredError, UnexpectedResponseError } from '../api/client';
import { signInPath } from '../lib/next';

/** History state the sign-in screen reads to say why it's showing. */
export interface SignInReason {
  reason: 'session-expired' | 'signed-out';
}

/** Sends the reader to sign in, and back here afterwards, once their session can't be refreshed. */
export function sessionExpiredHandler(router: DataRouter): () => void {
  return () => {
    const { pathname, search, hash } = router.state.location;
    if (pathname === '/sign-in') return;
    const state: SignInReason = { reason: 'session-expired' };
    void router.navigate(signInPath(pathname + search + hash), { replace: true, state });
  };
}

function isTransient(error: unknown): boolean {
  if (error instanceof NetworkError || error instanceof UnexpectedResponseError) return true;
  return error instanceof ProblemError && error.status >= 500;
}

interface QueryClientOptions {
  onSessionExpired: () => void;
  /** Tests turn retries off so failures show at once. */
  retry?: boolean;
}

export function createQueryClient({ onSessionExpired, retry = true }: QueryClientOptions): QueryClient {
  const onError = (error: unknown) => {
    if (error instanceof SessionExpiredError) onSessionExpired();
  };
  return new QueryClient({
    queryCache: new QueryCache({ onError }),
    mutationCache: new MutationCache({ onError }),
    defaultOptions: {
      queries: {
        retry: (failures, error) => retry && failures < 2 && isTransient(error),
      },
      mutations: { retry: false },
    },
  });
}
