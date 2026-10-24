import { describe, expect, it } from 'vitest';
import { NetworkError, ProblemError, SessionExpiredError, UnexpectedResponseError } from './client';
import { describeError, isProblem } from './describe';

function problemError(status: number, fields: { detail?: string; requestId?: string } = {}) {
  return new ProblemError({ status, title: 'Problem', ...fields });
}

describe('describeError', () => {
  it.each([
    ['a request that got no answer', new NetworkError('Failed to fetch'), { message: 'error.network', retryable: true }],
    ['an ended session', new SessionExpiredError('Session expired'), { message: 'error.sessionEnded', retryable: false }],
    [
      "an answer this app can't read",
      new UnexpectedResponseError(200, 'Unexpected body'),
      { message: 'error.unexpected', retryable: true },
    ],
    ['anything else thrown', new Error('boom'), { message: 'error.unknown', retryable: true }],
    ['too many attempts', problemError(429), { message: 'error.rateLimited', retryable: true }],
    ['a forbidden request', problemError(403), { message: 'error.forbidden', retryable: false }],
    ["something that isn't there", problemError(404), { message: 'error.notFound', retryable: false }],
  ])('describes %s', (_, error, described) => {
    expect(describeError(error)).toEqual(described);
  });

  it("gives a server fault's request ID to quote, but not its detail", () => {
    expect(describeError(problemError(503, { detail: 'connection refused', requestId: 'req-42' }))).toEqual({
      message: 'error.server',
      retryable: true,
      requestId: 'req-42',
    });
  });

  it("passes on the server's explanation of a request it refused", () => {
    expect(describeError(problemError(409, { detail: 'That address belongs to another account.' }))).toEqual({
      message: 'error.request',
      retryable: false,
      detail: 'That address belongs to another account.',
    });
  });
});

describe('isProblem', () => {
  it('matches a problem with one of the statuses only', () => {
    expect(isProblem(problemError(410), 404, 410)).toBe(true);
    expect(isProblem(problemError(500), 404, 410)).toBe(false);
    expect(isProblem(new NetworkError('Failed to fetch'), 404)).toBe(false);
  });
});
