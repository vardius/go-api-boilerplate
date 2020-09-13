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

export interface Token {
  id: string;
  access: string;
  refresh?: string;
  user_agent?: string;
}

export type AuthToken = string | null;
