/**
 * Returns where to go after signing in, as a path with its query and hash,
 * or undefined when `next` is missing or leads off this origin. Accepts a
 * path ("/apps") or an absolute URL on this origin, which the auth service
 * may send for its authorize endpoint. Everything else is dropped, so a
 * crafted link can't bounce someone to another site.
 */
export function safeNext(next: string | null | undefined, origin: string = window.location.origin): string | undefined {
  if (!next) return undefined;
  if (!next.startsWith('/') && !next.startsWith(`${origin}/`)) return undefined;
  let url: URL;
  try {
    url = new URL(next, origin);
  } catch {
    return undefined;
  }
  // Catches "//host", "/\host" and anything the URL parser normalises into another origin.
  if (url.origin !== origin) return undefined;
  // "/.//host" stays on this origin but normalises to the path "//host", which a link or navigation reads as a host.
  if (url.pathname.startsWith('//')) return undefined;
  return url.pathname + url.search + url.hash;
}

/** Paths served by the APIs (the OAuth authorize endpoint, for one) need a full page load, not a client-side route. */
export function isServerPath(path: string): boolean {
  return path.startsWith('/api/');
}

/** Builds the sign-in path that comes back to `next` afterwards. */
export function signInPath(next?: string): string {
  return next === undefined ? '/sign-in' : `/sign-in?${new URLSearchParams({ next }).toString()}`;
}
