import { env } from '$env/dynamic/private';
import { createApi } from '$lib/api';

/** Build an API client for use inside a SvelteKit server `load` or endpoint.
 *  Pass the event's `fetch`.
 *
 *  In production set `API_INTERNAL_URL` to the Go service origin (e.g.
 *  `http://app:8080`) so server-to-server calls reach the backend directly — a
 *  relative `/api` would hit this Node server, which has no such route. In dev
 *  it defaults to relative URLs, which the Vite proxy forwards to the backend.
 *
 *  Note: with an absolute base URL, `event.fetch` does not forward the request's
 *  cookies. That is fine for public reads; authenticated server-side calls
 *  (e.g. `/me`) get explicit cookie forwarding in a later task. */
export function serverApi(fetchImpl: typeof fetch) {
  return createApi(fetchImpl, env.API_INTERNAL_URL ?? '');
}
