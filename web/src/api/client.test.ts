import { describe, expect, it } from 'vitest';
import { z } from 'zod';
import { empty, json, mockApi, offline, problem } from '../test/api';
import {
  emptyBody,
  NetworkError,
  ProblemError,
  request,
  SessionExpiredError,
  UnexpectedResponseError,
} from './client';
import { userSchema } from './schemas';

const me = '/api/user/v1/me';
const refresh = '/api/auth/v1/sessions/refresh';
const user = {
  id: 'u1',
  email: 'ada@example.com',
  google_linked: false,
  created_at: '2026-10-24T08:15:30.123456789+11:00',
};

describe('request', () => {
  it('returns the body parsed by its schema', async () => {
    mockApi({ [`GET ${me}`]: json(200, user) });

    const result = await request(me, { schema: userSchema });

    expect(result.email).toBe('ada@example.com');
    // Nanoseconds and a +11:00 offset, as Go writes them, become the right instant.
    expect(result.created_at.toISOString()).toBe('2026-10-23T21:15:30.123Z');
  });

  it('sends JSON with same-origin credentials and never an Authorization header', async () => {
    const api = mockApi({ 'POST /api/user/v1/login-links': json(202, {}) });

    await request('/api/user/v1/login-links', {
      method: 'POST',
      body: { email: 'ada@example.com' },
      schema: emptyBody,
      session: false,
    });

    const [call] = api.calls;
    expect(call?.init?.credentials).toBe('same-origin');
    const headers = new Headers(call?.init?.headers);
    expect(headers.get('Content-Type')).toBe('application/json');
    expect(headers.has('Authorization')).toBe(false);
    expect(call?.body).toEqual({ email: 'ada@example.com' });
  });

  it('accepts an empty 204', async () => {
    mockApi({ 'DELETE /api/auth/v1/sessions/s1': empty(204) });

    await expect(
      request('/api/auth/v1/sessions/s1', { method: 'DELETE', schema: emptyBody }),
    ).resolves.toBeUndefined();
  });

  it('turns problem+json into a ProblemError with field errors and the request ID', async () => {
    mockApi({
      'POST /api/user/v1/login-links': problem(400, {
        title: 'Bad Request',
        detail: 'The request has invalid fields.',
        request_id: 'req-42',
        errors: { email: 'must be an email address' },
      }),
    });

    const error = await request('/api/user/v1/login-links', {
      method: 'POST',
      body: { email: 'nope' },
      schema: emptyBody,
      session: false,
    }).catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(ProblemError);
    expect(error).toMatchObject({
      status: 400,
      title: 'Bad Request',
      detail: 'The request has invalid fields.',
      requestId: 'req-42',
      fieldErrors: { email: 'must be an email address' },
    });
  });

  it('keeps the status when an error has no problem body, like a proxy page', async () => {
    mockApi({
      [`GET ${me}`]: () =>
        new Response('<html>Bad Gateway</html>', {
          status: 502,
          statusText: 'Bad Gateway',
          headers: { 'Content-Type': 'text/html', 'X-Request-Id': 'edge-7' },
        }),
    });

    const error = await request(me, { schema: userSchema }).catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(ProblemError);
    expect(error).toMatchObject({ status: 502, title: 'Bad Gateway', requestId: 'edge-7', fieldErrors: {} });
  });

  it('ignores a malformed errors map instead of failing the whole problem', async () => {
    mockApi({ [`GET ${me}`]: problem(400, { title: 'Bad Request', errors: ['not', 'a', 'map'] }) });

    const error = await request(me, { schema: userSchema }).catch((caught: unknown) => caught);

    expect(error).toMatchObject({ status: 400, title: 'Bad Request', fieldErrors: {} });
  });

  it('rejects a 2xx body that does not match the schema', async () => {
    mockApi({ [`GET ${me}`]: json(200, { ...user, google_linked: 'yes' }) });

    const error = await request(me, { schema: userSchema }).catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(UnexpectedResponseError);
    expect(String(error)).toContain('google_linked');
  });

  it('rejects a 2xx body that is not JSON', async () => {
    mockApi({ [`GET ${me}`]: () => new Response('<html></html>', { status: 200 }) });

    await expect(request(me, { schema: userSchema })).rejects.toBeInstanceOf(UnexpectedResponseError);
  });

  it('reports a failed fetch as a NetworkError and never tries to refresh', async () => {
    const api = mockApi({ [`GET ${me}`]: offline() });

    await expect(request(me, { schema: userSchema })).rejects.toBeInstanceOf(NetworkError);
    expect(api.callsTo(`POST ${refresh}`)).toHaveLength(0);
  });

  it('lets a cancelled request reject with its own abort, not a NetworkError', async () => {
    mockApi({ [`GET ${me}`]: () => Promise.reject(new DOMException('Aborted', 'AbortError')) });
    const controller = new AbortController();
    controller.abort();

    const error = await request(me, { schema: userSchema, signal: controller.signal }).catch(
      (caught: unknown) => caught,
    );

    expect(error).toBeInstanceOf(DOMException);
    expect(error).not.toBeInstanceOf(NetworkError);
  });
});

describe('request on 401', () => {
  it('refreshes the session once and retries', async () => {
    const api = mockApi({
      [`GET ${me}`]: [problem(401), json(200, user)],
      [`POST ${refresh}`]: empty(200),
    });

    const result = await request(me, { schema: userSchema });

    expect(result.id).toBe('u1');
    expect(api.calls.map((call) => `${call.method} ${call.path}`)).toEqual([
      `GET ${me}`,
      `POST ${refresh}`,
      `GET ${me}`,
    ]);
  });

  it('gives up with SessionExpiredError when the refresh is refused', async () => {
    const api = mockApi({ [`GET ${me}`]: problem(401), [`POST ${refresh}`]: problem(401) });

    await expect(request(me, { schema: userSchema })).rejects.toBeInstanceOf(SessionExpiredError);
    expect(api.callsTo(`GET ${me}`)).toHaveLength(1);
  });

  it('refreshes only once when the retry is refused too', async () => {
    const api = mockApi({ [`GET ${me}`]: problem(401), [`POST ${refresh}`]: empty(200) });

    await expect(request(me, { schema: userSchema })).rejects.toBeInstanceOf(SessionExpiredError);
    expect(api.callsTo(`POST ${refresh}`)).toHaveLength(1);
    expect(api.callsTo(`GET ${me}`)).toHaveLength(2);
  });

  it('shares one refresh between concurrent requests, since a rotated token may be used once', async () => {
    let release: () => void = () => undefined;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    const api = mockApi({
      [`GET ${me}`]: [problem(401), json(200, user)],
      'GET /api/auth/v1/sessions': [problem(401), json(200, { sessions: [] })],
      [`POST ${refresh}`]: async () => {
        await gate;
        return new Response(null, { status: 200 });
      },
    });

    const first = request(me, { schema: userSchema });
    const second = request('/api/auth/v1/sessions', { schema: z.object({ sessions: z.array(z.unknown()) }) });
    // Let both requests get their 401 and ask for a refresh while it is still in flight.
    await new Promise((resolve) => setTimeout(resolve, 0));
    release();

    await expect(Promise.all([first, second])).resolves.toHaveLength(2);
    expect(api.callsTo(`POST ${refresh}`)).toHaveLength(1);
  });

  it('reports an unreachable refresh as a NetworkError, not a lost session', async () => {
    mockApi({ [`GET ${me}`]: problem(401), [`POST ${refresh}`]: offline() });

    await expect(request(me, { schema: userSchema })).rejects.toBeInstanceOf(NetworkError);
  });

  it('reports a failing refresh endpoint as a ProblemError, not a lost session', async () => {
    mockApi({ [`GET ${me}`]: problem(401), [`POST ${refresh}`]: problem(503) });

    await expect(request(me, { schema: userSchema })).rejects.toMatchObject({ status: 503 });
  });

  it('treats a 401 from a public route as an answer, not an expired session', async () => {
    const api = mockApi({ 'POST /api/auth/v1/sessions': problem(401, { title: 'Unauthorized' }) });

    const error = await request('/api/auth/v1/sessions', {
      method: 'POST',
      body: { code: 'used' },
      schema: emptyBody,
      session: false,
    }).catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(ProblemError);
    expect(error).toMatchObject({ status: 401 });
    expect(api.callsTo(`POST ${refresh}`)).toHaveLength(0);
  });
});
