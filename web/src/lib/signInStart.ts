/**
 * Remembers that this browser started a sign-in, by email or with Google, so
 * the link or callback that finishes it can go straight in. A sign-in this
 * browser didn't start may be someone else's code, sent to sign the reader in
 * to the sender's account (login CSRF), so that one asks first.
 *
 * Only this app can redeem a code (the auth service refuses cross-origin
 * calls), and other sites can't write this origin's storage, so the mark
 * can't be forged.
 */
const storageKey = 'sign-in-started';

/** A login code works for 15 minutes (LoginCodeTTL in internal/user/app); a start older than that has nothing left to finish. */
const lifetimeMs = 15 * 60 * 1000;

export function markSignInStarted(now: number = Date.now()): void {
  try {
    window.localStorage.setItem(storageKey, String(now));
  } catch {
    // Storage is unavailable (private mode): the sign-in asks before it completes.
  }
}

/** Whether this browser started a sign-in in the last 15 minutes. */
export function signInStartedHere(now: number = Date.now()): boolean {
  try {
    const at = Number(window.localStorage.getItem(storageKey));
    return at > 0 && now >= at && now - at < lifetimeMs;
  } catch {
    return false;
  }
}

/** One start finishes one sign-in: called as its code is used, whatever the answer. */
export function forgetSignInStarted(): void {
  try {
    window.localStorage.removeItem(storageKey);
  } catch {
    // Storage is unavailable, so there is nothing to forget.
  }
}
