/**
 * How a Google sign-in or link that didn't finish ended, as the user service
 * puts it in ?google= on /sign-in or /profile (internal/user/app/app.go:
 * the Google* outcome constants).
 */
export const googleFailures = ['denied', 'expired', 'unverified', 'taken', 'failed'] as const;
export type GoogleFailure = (typeof googleFailures)[number];

/** The failure ?google= names, or undefined for anything else (including "linked"). */
export function googleFailure(value: string | null): GoogleFailure | undefined {
  return googleFailures.find((failure) => failure === value);
}
