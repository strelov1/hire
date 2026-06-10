// The only module that knows the API base URL and wire shapes. Views call the
// typed functions below; they never touch fetch or URLs directly. List endpoints
// return a `Slice` so callers (and the Paginator) stay ignorant of how each one
// signals more pages.

import type { Job, Company, CompanyListItem, ListMeta, User } from './types';

const BASE = (import.meta.env.VITE_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

/** A page of list items plus whether more remain after it. */
export interface Slice<T> {
  items: T[];
  hasMore: boolean;
}

interface Page<T> {
  data: T[];
  meta: ListMeta;
}

/** GET `path` and return the decoded JSON, throwing on network or non-2xx. */
async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

function query(limit: number, offset: number): string {
  return `?limit=${limit}&offset=${offset}`;
}

/** Turn a count-bearing page into a Slice; more remain unless we've reached total. */
function toSlice<T>(page: Page<T>, offset: number): Slice<T> {
  return { items: page.data, hasMore: offset + page.data.length < page.meta.total };
}

export async function listJobs(limit: number, offset: number): Promise<Slice<Job>> {
  return toSlice(await get<Page<Job>>(`/api/v1/jobs${query(limit, offset)}`), offset);
}

export async function getJob(id: string): Promise<Job> {
  const body = await get<{ data: Job }>(`/api/v1/jobs/${id}`);
  return body.data;
}

export async function listCompanies(limit: number, offset: number): Promise<Slice<CompanyListItem>> {
  return toSlice(await get<Page<CompanyListItem>>(`/api/v1/companies${query(limit, offset)}`), offset);
}

export async function getCompany(
  slug: string,
  limit: number,
  offset: number,
): Promise<{ company: Company; jobs: Job[] }> {
  const body = await get<{ data: { company: Company; jobs: Job[] } }>(
    `/api/v1/companies/${slug}${query(limit, offset)}`,
  );
  return body.data;
}

// --- Auth -------------------------------------------------------------------

/** A non-2xx API response. Carries the HTTP status so callers can branch on it
 *  (e.g. 401 invalid credentials, 409 email taken) instead of parsing strings. */
export class ApiError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

/** What register/login return: the user plus a freshly signed token. */
export interface AuthResult {
  user: User;
  token: string;
}

/** POST JSON and return the decoded `data`, throwing ApiError on non-2xx. */
async function postAuth(path: string, body: unknown): Promise<AuthResult> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new ApiError(res.status, `${res.status} ${res.statusText}`);
  }
  return ((await res.json()) as { data: AuthResult }).data;
}

export function register(email: string, password: string): Promise<AuthResult> {
  return postAuth('/api/v1/auth/register', { email, password });
}

export function login(email: string, password: string): Promise<AuthResult> {
  return postAuth('/api/v1/auth/login', { email, password });
}

/** Fetch the current user for a token. Throws ApiError(401) if it is rejected. */
export async function me(token: string): Promise<User> {
  const res = await fetch(`${BASE}/api/v1/auth/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    throw new ApiError(res.status, `${res.status} ${res.statusText}`);
  }
  return ((await res.json()) as { data: User }).data;
}
