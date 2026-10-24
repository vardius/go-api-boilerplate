import { z } from 'zod';

/**
 * An RFC 3339 timestamp as Go writes it, parsed to a Date. Go keeps up to nine
 * fractional digits; Date parsing is only specified for three, so the rest are
 * dropped first.
 */
const timestamp = z.iso
  .datetime({ offset: true })
  .transform((value) => new Date(value.replace(/(\.\d{3})\d+/, '$1')));

const httpURL = z.url({ protocol: /^https?$/ });

export const configSchema = z.object({
  google_enabled: z.boolean(),
  mail_preview_url: z.preprocess((value) => (value === '' || value === null ? undefined : value), httpURL.optional()),
});
export type Config = z.output<typeof configSchema>;

export const userSchema = z.object({
  id: z.string().min(1),
  email: z.string().min(1),
  google_linked: z.boolean(),
  created_at: timestamp,
});
export type User = z.output<typeof userSchema>;

export const createdSessionSchema = z.object({
  session: z.object({ id: z.string().min(1), expires_at: timestamp }),
});

export const sessionSchema = z.object({
  id: z.string().min(1),
  created_at: timestamp,
  last_used_at: timestamp.nullish(),
  user_agent: z.string().nullish(),
  current: z.boolean(),
});
export type Session = z.output<typeof sessionSchema>;

export const sessionsSchema = z.object({ sessions: z.array(sessionSchema) });

export const clientSchema = z.object({
  id: z.string().min(1),
  name: z.string(),
  redirect_uris: z.array(z.string()),
  scopes: z.array(z.string()),
  created_at: timestamp.optional(),
});
export type Client = z.output<typeof clientSchema>;

export const clientPageSchema = z.object({
  clients: z.array(clientSchema),
  limit: z.number().int().positive(),
  offset: z.number().int().nonnegative(),
});
export type ClientPage = z.output<typeof clientPageSchema>;

export const createdClientSchema = z.object({
  client: clientSchema,
  client_secret: z.string().min(1),
});
export type CreatedClient = z.output<typeof createdClientSchema>;

export const authorizationRequestSchema = z.object({
  client: z.object({ id: z.string().min(1), name: z.string() }),
  scopes: z.array(z.object({ name: z.string(), description: z.string() })),
  redirect_host: z.string().min(1),
});
export type AuthorizationRequest = z.output<typeof authorizationRequestSchema>;
