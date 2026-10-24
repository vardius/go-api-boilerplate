/** Leaves the single-page app for a server route. Kept apart so tests can observe the navigation. */
export function loadPage(path: string): void {
  window.location.assign(path);
}
