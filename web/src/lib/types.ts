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
  // Work arrangement.
  work_mode?: string;
  employment_type?: string;
  relocation?: string;
  visa_sponsorship?: boolean;

  // Location / eligibility. regions is a remote role's reach (meaningful only
  // when work_mode is remote): 'global' + macro-regions + select countries.
  // Empty = unknown; 'global' is explicit, distinct from unknown.
  regions?: string[];
  countries?: string[];
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

/** A signed-in user's interaction with one job: when they viewed it and, once
 *  they confirm an application, when they applied. `applied_at` is null until
 *  then. Returned by the view/apply endpoints. */
export interface UserJob {
  job_id: number;
  viewed_at: string;
  applied_at: string | null;
}
