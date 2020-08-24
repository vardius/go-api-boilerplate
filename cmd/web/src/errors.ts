export interface Options {
  statusCode: number;
}

export default class HttpError extends Error {
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
