// Wire types mirroring the backend JSON (internal/db models and query rows).
// Timestamps marshal as RFC3339 strings when present, or null.

export interface Job {
  id: number;
  source: string;
  external_id: string;
  url: string;
  title: string;
  company: string;
  company_slug: string;
  location: string;
  remote: boolean;
  description: string;
  posted_at: string | null;
  created_at: string | null;
  updated_at: string | null;
}

export interface Company {
  slug: string;
  name: string;
  created_at: string | null;
  updated_at: string | null;
}

/** A row of the companies catalog: company plus its computed job count. */
export interface CompanyListItem {
  slug: string;
  name: string;
  job_count: number;
}

/** Pagination metadata returned alongside list responses. */
export interface ListMeta {
  total: number;
  limit: number;
  offset: number;
}

/** An authenticated account, as returned by the auth endpoints. */
export interface User {
  id: number;
  email: string;
  created_at: string | null;
}
