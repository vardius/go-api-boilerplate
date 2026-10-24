import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { getAuthorizationRequest, getConfig, getMe, listClients, listSessions } from './endpoints';

export const queryKeys = {
  config: ['config'],
  me: ['me'],
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

export function useSessions() {
  return useQuery({ queryKey: queryKeys.sessions, queryFn: ({ signal }) => listSessions(signal) });
}

/** The caller's OAuth clients, a page at a time. A short page is the last one. */
export function useClients() {
  return useInfiniteQuery({
    queryKey: queryKeys.clients,
    queryFn: ({ pageParam, signal }) => listClients(pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (page) => (page.clients.length < page.limit ? undefined : page.offset + page.limit),
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
