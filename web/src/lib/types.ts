// Wire types mirroring the backend JSON (internal/db models and query rows).
// Timestamps marshal as RFC3339 strings when present, or null.

export interface Job {
  // Public, non-enumerable identifier used in URLs; the internal numeric id is
  // never sent to the client.
  public_slug: string;
  source: string;
  external_id: string;
  url: string;
  title: string;
  company: string;
  company_slug: string;
  location: string;
  description: string;
  posted_at: string | null;
  created_at: string | null;
  updated_at: string | null;
  // Non-null when the posting is no longer open. Lists never serve closed
  // jobs; the detail page renders the closed state from this field.
  closed_at: string | null;
  // Resolved geography facet, served top-level: the union of the parsed-location
  // columns and the enrichment-derived values. work_mode is the LLM value when
  // present, else the one parsed from the location. regions is the job's
  // geographic area (any work mode); empty = unknown, 'global' is explicit.
  regions?: string[];
  countries?: string[];
  work_mode?: string;
  enrichment?: Enrichment;
  enriched_at?: string | null;
  enrichment_version?: number;
}

/**
 * AI-derived, structured view of a job. Mirrors the Go `enrich.Enrichment`
 * contract (internal/enrich/enrichment.go) — snake_case keys, every field
 * optional and absent when the source did not state it. Enum values are
 * validated server-side before storage, so the SPA only formats them.
 */
export interface Enrichment {
  // Work arrangement. work_mode, regions, and countries are NOT here: they are
  // folded into the top-level Job geography facet (see Job above) and served once.
  employment_type?: string;
  relocation?: string;
  visa_sponsorship?: boolean;

  // Location / eligibility.
  cities?: string[];
  timezone_note?: string;

  // Compensation.
  salary_min?: number;
  salary_max?: number;
  salary_currency?: string;
  salary_period?: string;

  // Requirements / qualifications.
  seniority?: string;
  experience_years_min?: number;
  english_level?: string;
  education_level?: string;
  skills?: string[];

  // Classification.
  category?: string;
  domains?: string[];
  posting_language?: string;

  // Company descriptors.
  company_type?: string;
  company_size?: string;
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

/** A signed-in user's interaction with one job: when they viewed it, saved it
 *  for later, and (once they confirm an application) applied. `saved_at` and
 *  `applied_at` are null until then. Returned by the view/apply/save endpoints. */
export interface UserJob {
  job_id: number;
  viewed_at: string;
  saved_at: string | null;
  applied_at: string | null;
  // Application pipeline stage + free-text notes; null until set via `track`.
  stage: string | null;
  notes: string | null;
}

/** One item of the my-jobs listing: the job in the shared wire shape with the
 *  caller's interaction timestamps riding alongside. */
export interface MyJob {
  job: Job;
  viewed_at: string;
  saved_at: string | null;
  applied_at: string | null;
  stage: string | null;
  notes: string | null;
}

/** Per-tab row counts for the my-jobs page, from the listing's meta. `viewed`
 *  is the view-only subset: rows neither saved nor applied. */
export interface MyJobCounts {
  all: number;
  viewed: number;
  saved: number;
  applied: number;
}

/** An API key as returned by the management endpoints — metadata only; the
 *  plaintext token is never part of this shape. `token_prefix` is a short,
 *  non-secret leading slice (e.g. "fhk_Ab12cd") shown so the user can tell keys
 *  apart. Timestamps are RFC3339 strings or null. */
export interface ApiKey {
  id: number;
  name: string;
  token_prefix: string;
  created_at: string | null;
  last_used_at: string | null;
  expires_at: string | null;
}

/** The response of creating a key: the metadata plus the plaintext `token`,
 *  returned exactly once and never retrievable again. */
export interface CreatedApiKey extends ApiKey {
  token: string;
}
