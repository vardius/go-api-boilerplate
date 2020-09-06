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
}

export type AuthToken = string | null;
