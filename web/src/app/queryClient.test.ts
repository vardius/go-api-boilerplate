import { describe, expect, it, vi } from 'vitest';
import { NetworkError, ProblemError, SessionExpiredError, UnexpectedResponseError } from '../api/client';
import { createQueryClient } from './queryClient';

/** How many times a query failing with error runs before it gives up. */
async function attempts(error: Error, retry?: boolean) {
  const onSessionExpired = vi.fn();
  const queryClient = createQueryClient({ onSessionExpired, ...(retry === undefined ? {} : { retry }) });
  const queryFn = vi.fn(() => Promise.reject(error));

  await expect(queryClient.query({ queryKey: ['thing'], queryFn, retryDelay: 0 })).rejects.toBe(error);
  return { runs: queryFn.mock.calls.length, onSessionExpired };
}

describe('createQueryClient', () => {
  it.each([
    ['no answer', new NetworkError('Failed to fetch')],
    ["an answer this app can't read", new UnexpectedResponseError(200, 'Unexpected body')],
    ['a server fault', new ProblemError({ status: 503, title: 'Service Unavailable' })],
  ])('asks twice more after %s', async (_, error) => {
    expect((await attempts(error)).runs).toBe(3);
  });

  it.each([
    ['too many attempts', new ProblemError({ status: 429, title: 'Too Many Requests' })],
    ['a refused request', new ProblemError({ status: 404, title: 'Not Found' })],
  ])("doesn't ask again after %s", async (_, error) => {
    expect((await attempts(error)).runs).toBe(1);
  });

  it('sends the reader to sign in once the session has ended, without asking again', async () => {
    const { runs, onSessionExpired } = await attempts(new SessionExpiredError('Session expired'));

    expect(runs).toBe(1);
    expect(onSessionExpired).toHaveBeenCalledOnce();
  });

  it('asks once when retries are off', async () => {
    expect((await attempts(new NetworkError('Failed to fetch'), false)).runs).toBe(1);
  });
});
