// The only module that knows the API base URL and wire shapes. Views call the
// typed functions below; they never touch fetch or URLs directly. List endpoints
// return a `Slice` so callers (and the Paginator) stay ignorant of how each one
// signals more pages.

import type { Job, Company, CompanyListItem, ListMeta, User, UserJob } from './types';

// Relative base: the SPA and API share one origin (a dev Vite proxy forwards
// /api to the backend), so the browser sends the httpOnly auth cookie with
// every request. No absolute URL, no CORS.
const BASE = '';

/** A page of list items, optionally the total matching the query (endpoints that
 *  report one), and whether more remain. */
export interface Slice<T> {
  items: T[];
  total?: number;
  hasMore: boolean;
}

interface Page<T> {
  data: T[];
  meta: ListMeta;
}

/** A non-2xx API response. Carries the HTTP status so callers can branch on it
 *  (e.g. 401 invalid credentials, 409 email taken) instead of parsing strings. */
export class ApiError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

/** The single place this module touches fetch. Always sends credentials so the
 *  auth cookie rides along, and turns a non-2xx into an ApiError. */
async function call(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(`${BASE}${path}`, { credentials: 'include', ...init });
  if (!res.ok) {
    throw new ApiError(res.status, `${res.status} ${res.statusText}`);
  }
  return res;
}

/** Call `path` and return the decoded JSON body. A bare call (no init) is a GET. */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  return (await call(path, init)).json() as Promise<T>;
}

function query(limit: number, offset: number): string {
  return `?limit=${limit}&offset=${offset}`;
}

/** Turn a count-bearing page into a Slice; more remain unless we've reached total. */
function toSlice<T>(page: Page<T>, offset: number): Slice<T> {
  return {
    items: page.data,
    total: page.meta.total,
    hasMore: offset + page.data.length < page.meta.total,
  };
}

export async function listJobs(limit: number, offset: number): Promise<Slice<Job>> {
  return toSlice(await request<Page<Job>>(`/api/v1/jobs${query(limit, offset)}`), offset);
}

export async function getJob(slug: string): Promise<Job> {
  const body = await request<{ data: Job }>(`/api/v1/jobs/${slug}`);
  return body.data;
}

/** Full-text search over jobs. `facets` carries the query text and any facet
 *  filters (built by the caller); pagination is appended here. Results are the
 *  same Job wire shape as listJobs, so views render them with the same
 *  components. `meta.total` is an estimate from the search engine.
 *
 *  Keyword-only by default (semantic_ratio=0): hybrid/semantic ranking scores
 *  every job by similarity, so a query like "devops" returns the whole catalogue
 *  reordered rather than the handful that match — which reads as "search is
 *  broken". Semantic stays available on the API for an explicit opt-in later. */
export async function searchJobs(facets: URLSearchParams, limit: number, offset: number): Promise<Slice<Job>> {
  const params = new URLSearchParams(facets);
  params.set('semantic_ratio', '0');
  params.set('limit', String(limit));
  params.set('offset', String(offset));
  return toSlice(await request<Page<Job>>(`/api/v1/jobs/search?${params}`), offset);
}

/** List companies, optionally filtered by a name query `q` (a case-insensitive
 *  substring match; an empty `q` lists everything). `meta.total` reflects the
 *  filtered count, so the Paginator pages over the matches. */
export async function listCompanies(q: string, limit: number, offset: number): Promise<Slice<CompanyListItem>> {
  const params = new URLSearchParams();
  if (q) params.set('q', q);
  params.set('limit', String(limit));
  params.set('offset', String(offset));
  return toSlice(await request<Page<CompanyListItem>>(`/api/v1/companies?${params}`), offset);
}

export async function getCompany(
  slug: string,
  limit: number,
  offset: number,
): Promise<{ company: Company; jobs: Job[] }> {
  const body = await request<{ data: { company: Company; jobs: Job[] } }>(
    `/api/v1/companies/${slug}${query(limit, offset)}`,
  );
  return body.data;
}

// --- Auth -------------------------------------------------------------------
//
// register/login set the httpOnly auth cookie server-side and return the user;
// the token never reaches JS. Subsequent calls (me) are authenticated by the
// cookie the browser attaches automatically.

/** POST credentials and return the created/authenticated user. */
async function postAuth(path: string, body: unknown): Promise<User> {
  const res = await request<{ data: User }>(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  return res.data;
}

export function register(email: string, password: string): Promise<User> {
  return postAuth('/api/v1/auth/register', { email, password });
}

export function login(email: string, password: string): Promise<User> {
  return postAuth('/api/v1/auth/login', { email, password });
}

/** Clear the session cookie server-side. */
export async function logout(): Promise<void> {
  await call('/api/v1/auth/logout', { method: 'POST' });
}

/** Fetch the current user using the auth cookie. Throws ApiError(401) if it is
 *  missing or rejected. */
export async function me(): Promise<User> {
  const res = await request<{ data: User }>('/api/v1/auth/me');
  return res.data;
}

// --- Per-user job interactions ----------------------------------------------
//
// Both require a session (the auth cookie). Callers gate on auth state before
// invoking — the SPA never sends these for a signed-out visitor.

/** POST to a job-interaction endpoint and return the resulting record. */
async function postJobInteraction(slug: string, action: 'view' | 'apply'): Promise<UserJob> {
  const res = await request<{ data: UserJob }>(`/api/v1/jobs/${slug}/${action}`, { method: 'POST' });
  return res.data;
}

/** Record that the current user viewed a job; returns their interaction
 *  (including whether they have already applied). */
export function recordJobView(slug: string): Promise<UserJob> {
  return postJobInteraction(slug, 'view');
}

/** Mark a job as applied for the current user. */
export function markJobApplied(slug: string): Promise<UserJob> {
  return postJobInteraction(slug, 'apply');
}
