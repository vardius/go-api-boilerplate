import { createContext, use } from 'react';
import { isLocale, pickLocale, type Locale } from './messages';

export interface LocaleState {
  locale: Locale;
  setLocale: (locale: Locale) => void;
}

export const LocaleContext = createContext<LocaleState | null>(null);

export function useLocale(): LocaleState {
  const state = use(LocaleContext);
  if (state === null) throw new Error('useLocale needs a LocaleProvider above it.');
  return state;
}

const storageKey = 'locale';

/** The reader's saved choice, else their browser's languages. Storage may be blocked (private mode), which is fine. */
export function initialLocale(): Locale {
  try {
    const saved = window.localStorage.getItem(storageKey);
    if (isLocale(saved)) return saved;
  } catch {
    // Storage is unavailable; fall back to the browser's languages.
  }
  return pickLocale(navigator.languages);
}

export function saveLocale(locale: Locale): void {
  try {
    window.localStorage.setItem(storageKey, locale);
  } catch {
    // Storage is unavailable; the choice lasts until the page reloads.
  }
}
