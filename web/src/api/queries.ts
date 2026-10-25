import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { getAuthorizationRequest, getConfig, getMe, getSignedInUser, listClients, listSessions } from './endpoints';

export const queryKeys = {
  config: ['config'],
  me: ['me'],
  signedIn: ['signed-in'],
  sessions: ['sessions'],
  clients: ['clients'],
  authorizationRequest: (id: string) => ['authorization-request', id],
} as const;

/** Server feature flags. They change only on a redeploy. */
export function useConfig() {
  return useQuery({
    queryKey: queryKeys.config,
    queryFn: ({ signal }) => getConfig(signal),
    staleTime: Infinity,
  });
}

export function useMe() {
  return useQuery({ queryKey: queryKeys.me, queryFn: ({ signal }) => getMe(signal) });
}

/** Who this browser is signed in as right now, or null. Checked afresh each time, never from cache. */
export function useSignedInUser(options: { enabled: boolean }) {
  return useQuery({
    queryKey: queryKeys.signedIn,
    queryFn: ({ signal }) => getSignedInUser(signal),
    enabled: options.enabled,
    staleTime: 0,
    gcTime: 0,
    refetchOnWindowFocus: false,
  });
}

export function useSessions() {
  return useQuery({ queryKey: queryKeys.sessions, queryFn: ({ signal }) => listSessions(signal) });
}

/** The caller's OAuth clients, a page at a time. A page without a next cursor is the last one. */
export function useClients() {
  return useInfiniteQuery({
    queryKey: queryKeys.clients,
    queryFn: ({ pageParam, signal }) => listClients(pageParam, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page) => page.next_cursor,
  });
}

export function useAuthorizationRequest(id: string) {
  return useQuery({
    queryKey: queryKeys.authorizationRequest(id),
    queryFn: ({ signal }) => getAuthorizationRequest(id, signal),
    // Consent details don't change while the page is open, and a refetch could outlive the request.
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });
}
