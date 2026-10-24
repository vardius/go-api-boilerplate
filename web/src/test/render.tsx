import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter } from 'react-router';
import { App } from '../app/App';
import { createQueryClient, sessionExpiredHandler } from '../app/queryClient';
import { routes } from '../app/routes';

/** Renders the whole app at `path`, as a browser would, with the real routes and query client. */
export function renderApp(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path] });
  const queryClient = createQueryClient({ onSessionExpired: sessionExpiredHandler(router), retry: false });
  const user = userEvent.setup();
  const view = render(<App router={router} queryClient={queryClient} locale="en" />);
  return { ...view, router, queryClient, user };
}

/** Builds a user as the API returns it. */
export function apiUser(overrides: Record<string, unknown> = {}) {
  return {
    id: '0192f0c1-7b6e-7c3a-9a51-3f1d2a4b5c6d',
    email: 'ada@example.com',
    google_linked: false,
    created_at: '2026-01-15T09:30:00.123456789Z',
    ...overrides,
  };
}
