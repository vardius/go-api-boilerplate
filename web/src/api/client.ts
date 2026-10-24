import { z } from 'zod';

/** An RFC 9457 problem the API answered with, or a non-2xx response that had none. */
export class ProblemError extends Error {
  override readonly name = 'ProblemError';
  readonly status: number;
  readonly title: string;
  readonly detail: string | undefined;
  readonly requestId: string | undefined;
  /** Validation problems by request field, e.g. `{ email: "must be an email address" }`. */
  readonly fieldErrors: Readonly<Record<string, string>>;

  constructor(problem: {
    status: number;
    title: string;
    detail?: string | undefined;
    requestId?: string | undefined;
    fieldErrors?: Record<string, string> | undefined;
  }) {
    super(problem.detail ?? problem.title);
    this.status = problem.status;
    this.title = problem.title;
    this.detail = problem.detail;
    this.requestId = problem.requestId;
    this.fieldErrors = problem.fieldErrors ?? {};
  }
}

/** The request never got an answer: offline, DNS, connection reset. Retrying may help; it never means signed out. */
export class NetworkError extends Error {
  override readonly name = 'NetworkError';
}

/** The session is gone: the API said 401 and refreshing it once didn't help. */
export class SessionExpiredError extends Error {
  override readonly name = 'SessionExpiredError';
}

/** A 2xx response whose body isn't the shape this app was built against. */
export class UnexpectedResponseError extends Error {
  override readonly name = 'UnexpectedResponseError';
  readonly status: number;

  constructor(status: number, message: string, options?: ErrorOptions) {
    super(message, options);
    this.status = status;
  }
}

/** Matches a response with no body (204, or 202 with `{}`). */
export const emptyBody = z.union([z.undefined(), z.object({})]).transform(() => undefined);

const problemSchema = z.object({
  title: z.string().optional(),
  detail: z.string().optional(),
  request_id: z.string().optional(),
  errors: z.record(z.string(), z.string()).optional().catch(undefined),
});

const refreshPath = '/api/auth/v1/sessions/refresh';

export interface RequestOptions<S extends z.ZodType> {
  method?: 'GET' | 'POST' | 'DELETE';
  /** Sent as JSON. */
  body?: unknown;
  schema: S;
  /**
   * Whether the route needs a session. A 401 from such a route refreshes the
   * session once and retries; set false for public routes, where a 401 is an
   * ordinary answer (a bad sign-in code, for one).
   */
  session?: boolean;
  signal?: AbortSignal | undefined;
}

/**
 * Calls a same-origin API route and returns its body, validated by `schema`.
 * Credentials travel only as the HttpOnly cookies the server set.
 *
 * Throws ProblemError for an error status, NetworkError when there was no
 * answer, SessionExpiredError when the session can't be refreshed, and
 * UnexpectedResponseError when the body doesn't match `schema`.
 */
export async function request<S extends z.ZodType>(
  path: string,
  options: RequestOptions<S>,
): Promise<z.output<S>> {
  let response = await send(path, options);
  if (response.status === 401 && options.session !== false) {
    await refreshSession();
    response = await send(path, options);
    if (response.status === 401) throw new SessionExpiredError('The session was refreshed but is still refused.');
  }
  if (!response.ok) throw await toProblem(response);
  return parseBody(response, options.schema);
}

let refreshing: Promise<void> | undefined;

/**
 * Rotates the session cookies. Concurrent callers share one request, and tabs
 * take turns through a Web Lock: the server revokes the whole session when a
 * rotated refresh token is presented twice.
 */
export function refreshSession(): Promise<void> {
  refreshing ??= withRefreshLock(postRefresh).finally(() => {
    refreshing = undefined;
  });
  return refreshing;
}

function withRefreshLock(fn: () => Promise<void>): Promise<void> {
  if (!('locks' in navigator)) return fn();
  return navigator.locks.request('session-refresh', fn);
}

async function postRefresh(): Promise<void> {
  const response = await send(refreshPath, { method: 'POST' });
  if (response.status === 401) throw new SessionExpiredError('The session could not be refreshed.');
  if (!response.ok) throw await toProblem(response);
}

async function send(
  path: string,
  { method = 'GET', body, signal }: Pick<RequestOptions<z.ZodType>, 'method' | 'body' | 'signal'>,
): Promise<Response> {
  const headers = new Headers({ Accept: 'application/json, application/problem+json' });
  const init: RequestInit = { method, headers, credentials: 'same-origin' };
  if (body !== undefined) {
    headers.set('Content-Type', 'application/json');
    init.body = JSON.stringify(body);
  }
  if (signal) init.signal = signal;
  try {
    return await fetch(path, init);
  } catch (error) {
    // A cancelled query is not a failure; let the caller see its own abort.
    if (signal?.aborted) throw error;
    throw new NetworkError(`${method} ${path} got no response.`, { cause: error });
  }
}

async function readText(response: Response): Promise<string> {
  try {
    return await response.text();
  } catch (error) {
    throw new NetworkError('The response was cut off.', { cause: error });
  }
}

async function toProblem(response: Response): Promise<ProblemError> {
  const text = await readText(response);
  const parsed = problemSchema.safeParse(parseJSON(text));
  // A proxy's HTML error page has no problem body; the status still says what happened.
  const problem = parsed.success ? parsed.data : {};
  return new ProblemError({
    status: response.status,
    title: problem.title ?? response.statusText,
    detail: problem.detail,
    requestId: problem.request_id ?? response.headers.get('X-Request-Id') ?? undefined,
    fieldErrors: problem.errors,
  });
}

function parseJSON(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return undefined;
  }
}

async function parseBody<S extends z.ZodType>(response: Response, schema: S): Promise<z.output<S>> {
  const text = await readText(response);
  let data: unknown;
  if (text !== '') {
    try {
      data = JSON.parse(text);
    } catch (error) {
      throw new UnexpectedResponseError(response.status, 'The response is not JSON.', { cause: error });
    }
  }
  const result = schema.safeParse(data);
  if (!result.success) {
    throw new UnexpectedResponseError(response.status, z.prettifyError(result.error), { cause: result.error });
  }
  return result.data;
}
