import { QueryClientProvider, type QueryClient } from '@tanstack/react-query';
import type { DataRouter } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { LocaleProvider } from '../i18n/LocaleProvider';
import type { Locale } from '../i18n/messages';

interface AppProps {
  router: DataRouter;
  queryClient: QueryClient;
  locale?: Locale;
}

export function App({ router, queryClient, locale }: AppProps) {
  return (
    <LocaleProvider {...(locale === undefined ? {} : { locale })}>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </LocaleProvider>
  );
}
