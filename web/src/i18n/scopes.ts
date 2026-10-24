import type { MessageId } from './messages';

/** The scopes an OAuth app may register, per the auth service's grantable set. */
export const grantableScopes = ['user:read', 'user:write', 'clients:read', 'clients:write'] as const;
export type GrantableScope = (typeof grantableScopes)[number];

export const scopeDescriptions: Record<GrantableScope, MessageId> = {
  'user:read': 'scope.user:read',
  'user:write': 'scope.user:write',
  'clients:read': 'scope.clients:read',
  'clients:write': 'scope.clients:write',
};

/** The translated description of a known scope; unknown scopes fall back to the server's text. */
export function scopeDescriptionId(scope: string): MessageId | undefined {
  const known = grantableScopes.find((name) => name === scope);
  return known === undefined ? undefined : scopeDescriptions[known];
}
