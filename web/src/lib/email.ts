import type { MessageId } from '../i18n/messages';

/** A light check before sending; the server has the final word and answers 400 with a field error. */
export function emailProblem(email: string): MessageId | undefined {
  if (email === '') return 'signIn.emailRequired';
  if (!/^[^\s@]+@[^\s@]+$/.test(email)) return 'signIn.emailInvalid';
  return undefined;
}
