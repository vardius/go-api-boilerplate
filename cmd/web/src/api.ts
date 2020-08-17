import {API_URL} from "src/constants";

interface Options {
  statusCode: number;
}

export class HttpError extends Error {
  public readonly statusCode: number;

  constructor(message?: string, options?: Options) {
    super(message);

    const {statusCode} = options || {};
    this.statusCode = statusCode || 500;
  }
}

export class NotFoundHttpError extends HttpError {
  constructor(message?: string) {
    super(message, {statusCode: 404});
  }
}

export class UnauthorizedHttpError extends HttpError {
  constructor(message?: string) {
    super(message, {statusCode: 401});
  }
}

export class AccessDeniedHttpError extends HttpError {
  constructor(message?: string) {
    super(message, {statusCode: 403});
  }
}

export const fetchJSON = async (
  path: string,
  method: string,
  params?: { [key: string]: string | Array<string> } | null,
  body?: string
): Promise<any> => {
  const url = new URL(decodeURIComponent(path), API_URL);

  if (params) {
    Object.keys(params).forEach((k) => {
      if (Array.isArray(params[k])) {
        (params[k] as Array<string>).forEach((v) => {
          url.searchParams.set(k, v);
        });
      } else {
        url.searchParams.set(k, params[k] as string);
      }
    });
  }

  const response = await fetch(url.toString(), {
    method,
    headers: {
      "Content-Type": "application/json",
    },
    body,
  });

  if (!response.ok) {
    switch (response.status) {
      case 401:
        throw new UnauthorizedHttpError(response.statusText);
      case 403:
        throw new AccessDeniedHttpError(response.statusText);
      case 404:
        throw new NotFoundHttpError(response.statusText);
      default:
        throw new HttpError(response.statusText);
    }
  }

  return response.json();
};
