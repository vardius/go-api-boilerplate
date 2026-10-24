import { vi } from 'vitest';

type Reply = () => Response | Promise<Response>;

export interface Call {
  method: string;
  path: string;
  body: unknown;
  init: RequestInit | undefined;
}

export function json(status: number, body: unknown, headers: Record<string, string> = {}): Reply {
  return () =>
    new Response(JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json', ...headers },
    });
}

export function problem(status: number, fields: Record<string, unknown> = {}): Reply {
  return () =>
    new Response(JSON.stringify({ type: 'about:blank', title: 'Problem', status, ...fields }), {
      status,
      headers: { 'Content-Type': 'application/problem+json' },
    });
}

export function empty(status = 204): Reply {
  return () => new Response(null, { status });
}

/** fetch rejecting the way browsers do when the server can't be reached. */
export function offline(): Reply {
  return () => Promise.reject(new TypeError('Failed to fetch'));
}

/**
 * Stubs global fetch with canned replies per "METHOD /path". A route given
 * several replies uses them in order and repeats the last. Unrouted requests
 * fail the test loudly.
 */
export function mockApi(routes: Record<string, Reply | Reply[]> = {}) {
  const table = new Map<string, Reply[]>();
  const calls: Call[] = [];

  const on = (route: string, ...replies: Reply[]) => {
    table.set(route, replies);
  };
  for (const [route, replies] of Object.entries(routes)) on(route, ...(Array.isArray(replies) ? replies : [replies]));

  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const method = init?.method ?? 'GET';
    const path = typeof input === 'string' ? input : input instanceof URL ? input.pathname + input.search : input.url;
    const text = typeof init?.body === 'string' ? init.body : undefined;
    calls.push({ method, path, body: text === undefined ? undefined : JSON.parse(text), init });
    const replies = table.get(`${method} ${path}`);
    if (replies === undefined || replies.length === 0) throw new Error(`Unexpected request: ${method} ${path}`);
    const reply = replies.length > 1 ? replies.shift() : replies[0];
    if (reply === undefined) throw new Error(`No reply for ${method} ${path}`);
    return reply();
  });
  vi.stubGlobal('fetch', fetchMock);

  return {
    on,
    calls,
    /** The requests made to one route, in order. */
    callsTo: (route: string) => calls.filter((call) => `${call.method} ${call.path}` === route),
  };
}
