import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { IntlProvider } from 'react-intl';
import { initialLocale, LocaleContext, saveLocale, type LocaleState } from './locale';
import { messages, type Locale } from './messages';

interface LocaleProviderProps {
  /** Overrides detection; tests pin a locale with it. */
  locale?: Locale;
  children: ReactNode;
}

export function LocaleProvider({ locale: fixed, children }: LocaleProviderProps) {
  const [locale, setLocaleState] = useState<Locale>(() => fixed ?? initialLocale());

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const state = useMemo<LocaleState>(
    () => ({
      locale,
      setLocale: (next) => {
        setLocaleState(next);
        saveLocale(next);
      },
    }),
    [locale],
  );

  return (
    <LocaleContext value={state}>
      <IntlProvider locale={locale} defaultLocale="en" messages={messages[locale]}>
        {children}
      </IntlProvider>
    </LocaleContext>
  );
}
