import { z } from 'zod';

/**
 * Lets this browser recognise its own sign-in links. Starting a sign-in, by
 * email or with Google, makes a random state that the user service puts in
 * the link it sends back, and the link finishes at once only when its state
 * is one this browser made. Any other link may be someone else's code, sent
 * to sign the reader in to the sender's account (login CSRF), so that one
 * asks first.
 *
 * Other sites can't read this origin's storage, so they can't learn a
 * state, and only this app can redeem a code: the auth service refuses
 * cross-origin calls.
 */
const storageKey = 'sign-in-states';

/** A login code works for 15 minutes (LoginCodeTTL in internal/user/app); a state older than that has nothing left to finish. */
const lifetimeMs = 15 * 60 * 1000;

/** Enough for a few links asked for in a row; the oldest goes first. */
const kept = 5;

const startedSchema = z.array(z.object({ state: z.string(), at: z.number() }));
type Started = z.infer<typeof startedSchema>[number];

function load(now: number): Started[] {
  try {
    const stored: unknown = JSON.parse(window.localStorage.getItem(storageKey) ?? '[]');
    const parsed = startedSchema.safeParse(stored);
    if (!parsed.success) return [];
    return parsed.data.filter((started) => now >= started.at && now - started.at < lifetimeMs);
  } catch {
    return [];
  }
}

function save(started: Started[]): void {
  try {
    window.localStorage.setItem(storageKey, JSON.stringify(started.slice(-kept)));
  } catch {
    // Storage is unavailable (private mode): every link asks before it signs in.
  }
}

/** A new state for one sign-in: 16 random bytes, base64url, as the user service accepts. */
export function newSignInState(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return btoa(String.fromCharCode(...bytes))
    .replaceAll('+', '-')
    .replaceAll('/', '_')
    .replace(/=+$/, '');
}

/** Remembers state as one this browser started a sign-in with. */
export function rememberSignInState(state: string, now: number = Date.now()): void {
  save([...load(now), { state, at: now }]);
}

/** Whether state is one this browser started a sign-in with in the last 15 minutes. */
export function isOwnSignInState(state: string | null, now: number = Date.now()): boolean {
  return state !== null && state !== '' && load(now).some((started) => started.state === state);
}

/** One state finishes one sign-in: forgotten as its link's code is used, whatever the answer. */
export function forgetSignInState(state: string | null, now: number = Date.now()): void {
  save(load(now).filter((started) => started.state !== state));
}
