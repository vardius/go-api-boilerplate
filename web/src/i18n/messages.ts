import { en } from './en';
import { pl } from './pl';

export type MessageId = keyof typeof en;

export const locales = ['en', 'pl'] as const;
export type Locale = (typeof locales)[number];

export const messages: Record<Locale, Record<MessageId, string>> = { en, pl };

export function isLocale(value: unknown): value is Locale {
  return locales.some((locale) => locale === value);
}

/** The first of the reader's languages this app speaks, else English. */
export function pickLocale(languages: readonly string[]): Locale {
  for (const language of languages) {
    const base = language.toLowerCase().split('-')[0];
    if (isLocale(base)) return base;
  }
  return 'en';
}

declare global {
  // react-intl reads these to type message IDs and locales across the program.
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace FormatjsIntl {
    interface Message {
      ids: MessageId;
    }
    interface IntlConfig {
      locale: Locale;
    }
  }
}
