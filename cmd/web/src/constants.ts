import {LOCALE} from "./types";

export const DEFAULT_LOCALE = LOCALE.en;

export const AUTH_TOKEN_COOKIE = "at";
export const API_URL = window.location.hostname === 'localhost' ? 'https://api.go-api-boilerplate.local' :`https://api.${window.location.hostname}`;
export const DEFAULT_PAGE = 1;
export const DEFAULT_LIMIT = 10;
