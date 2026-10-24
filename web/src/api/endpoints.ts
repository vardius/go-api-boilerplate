import { emptyBody, request, SessionExpiredError } from './client';
import {
  authorizationRequestSchema,
  clientPageSchema,
  configSchema,
  createdClientSchema,
  createdSessionSchema,
  sessionsSchema,
  userSchema,
} from './schemas';

const auth = '/api/auth/v1';
const user = '/api/user/v1';

export function getConfig(signal?: AbortSignal) {
  return request(`${user}/config`, { schema: configSchema, session: false, signal });
}

/** state is this browser's, from lib/signInStart: the link carries it back. */
export function requestLoginLink(email: string, next: string | undefined, state: string) {
  return request(`${user}/login-links`, {
    method: 'POST',
    body: next === undefined ? { email, state } : { email, next, state },
    schema: emptyBody,
    session: false,
  });
}

/** Trades a one-time code from a sign-in link for session cookies. A bad, used or expired code is a 401 problem. */
export function createSession(code: string) {
  return request(`${auth}/sessions`, {
    method: 'POST',
    body: { code },
    schema: createdSessionSchema,
    session: false,
  });
}

/** Ends this browser's session. Without a session cookie the server answers 401, which means the same thing. */
export function deleteCurrentSession() {
  return request(`${auth}/sessions/current`, { method: 'DELETE', schema: emptyBody, session: false });
}

export function getMe(signal?: AbortSignal) {
  return request(`${user}/me`, { schema: userSchema, signal });
}

/**
 * The user this browser is signed in as, or null when it has no session (none,
 * or one that can't be refreshed). Unlike getMe, no session is an answer here,
 * not a reason to go to sign-in.
 */
export async function getSignedInUser(signal?: AbortSignal) {
  try {
    return await getMe(signal);
  } catch (error) {
    if (error instanceof SessionExpiredError) return null;
    throw error;
  }
}

export function requestEmailChange(email: string) {
  return request(`${user}/me/email-change`, { method: 'POST', body: { email }, schema: emptyBody });
}

export function confirmEmailChange(code: string) {
  return request(`${user}/me/email-change/confirm`, { method: 'POST', body: { code }, schema: userSchema });
}

export async function listSessions(signal?: AbortSignal) {
  const { sessions } = await request(`${auth}/sessions`, { schema: sessionsSchema, signal });
  return sessions;
}

export function revokeSession(id: string) {
  return request(`${auth}/sessions/${encodeURIComponent(id)}`, { method: 'DELETE', schema: emptyBody });
}

export function listClients(offset: number, signal?: AbortSignal) {
  const query = offset > 0 ? `?offset=${offset}` : '';
  return request(`${auth}/clients${query}`, { schema: clientPageSchema, signal });
}

export interface NewClient {
  name: string;
  redirect_uris: string[];
  scopes: string[];
}

export function createClient(client: NewClient) {
  return request(`${auth}/clients`, { method: 'POST', body: client, schema: createdClientSchema });
}

export function deleteClient(id: string) {
  return request(`${auth}/clients/${encodeURIComponent(id)}`, { method: 'DELETE', schema: emptyBody });
}

export function getAuthorizationRequest(id: string, signal?: AbortSignal) {
  return request(`${auth}/authorization-requests/${encodeURIComponent(id)}`, {
    schema: authorizationRequestSchema,
    signal,
  });
}

/** Where a browser starts Google sign-in; a full navigation, since it redirects to Google. */
export function googleStartURL(options: { next?: string | undefined; link?: boolean; state?: string }): string {
  const params = new URLSearchParams();
  if (options.link) params.set('link', '1');
  if (options.next !== undefined) params.set('next', options.next);
  if (options.state !== undefined) params.set('state', options.state);
  const query = params.toString();
  return `${user}/google/start${query === '' ? '' : `?${query}`}`;
}

/** The OAuth consent form posts here as a real form: the answer is a redirect to the app's site. */
export const authorizeFormAction = '/api/auth/oauth/authorize';
