export enum LOCALE {
  en = "en",
  pl = "pl",
}

export interface User {
  id?: string;
  email: string;
}

export type AuthToken = string | null;
