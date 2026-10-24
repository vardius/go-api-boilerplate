export enum LOCALE {
  en = "en",
  pl = "pl",
}

export interface User {
  id?: string;
  email: string;
}

export interface Client {
  id: string;
  secret: string;
  domain: string;
  redirect_url: string;
  scopes: Array<string>;
}

// Token is a session as the API lists it: never the token itself.
export interface Token {
  id: string;
  client_id: string;
  user_agent?: string;
  expires_at?: string;
}

export type AuthToken = string | null;
