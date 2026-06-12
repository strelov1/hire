// Presentation helpers for a job's AI enrichment. Pure functions that turn the
// controlled-vocabulary codes (validated server-side) into display labels.
// Unknown codes fall back to a humanized form so a future vocabulary addition
// never renders blank — the SPA never re-validates, it only formats.

import type { Enrichment } from './types';

/** A single label/value pair in the summary meta-row. */
export interface Facet {
  label: string;
  value: string;
}

const SENIORITY: Record<string, string> = {
  intern: 'Intern',
  junior: 'Junior',
  middle: 'Middle',
  senior: 'Senior',
  lead: 'Lead',
  principal: 'Principal',
  c_level: 'C-level',
};

const EMPLOYMENT: Record<string, string> = {
  full_time: 'Full-time',
  part_time: 'Part-time',
  contract: 'Contract',
  internship: 'Internship',
};

const WORK_MODE: Record<string, string> = {
  remote: 'Remote',
  hybrid: 'Hybrid',
  onsite: 'On-site',
};

// Reach codes (enrichment.regions) → readable labels. Unmapped codes humanize.
const REGION: Record<string, string> = {
  global: 'Global',
  eu: 'Europe',
  emea: 'EMEA',
  eea: 'EEA',
  uk: 'UK',
  americas: 'Americas',
  north_america: 'North America',
  latam: 'LATAM',
  apac: 'APAC',
  mena: 'MENA',
  africa: 'Africa',
  us: 'USA',
  ru: 'Russia',
};

const RELOCATION: Record<string, string> = {
  not_supported: 'Not supported',
  supported: 'Supported',
  required: 'Required',
};

const ENGLISH_LEVEL: Record<string, string> = {
  a1: 'A1',
  a2: 'A2',
  b1: 'B1',
  b2: 'B2',
  c1: 'C1',
  c2: 'C2',
  native: 'Native',
};

// Only the acronym/compound domains need a label; the rest (Crypto, Healthcare,
// Media…) humanize cleanly, so they fall through to `humanize` like CATEGORY.
const DOMAINS: Record<string, string> = {
  fintech: 'FinTech',
  ecommerce: 'E-commerce',
  saas: 'SaaS',
  gamedev: 'GameDev',
  edtech: 'EdTech',
  adtech: 'AdTech',
  govtech: 'GovTech',
};

const CATEGORY: Record<string, string> = {
  ml_ai: 'ML / AI',
  qa: 'QA',
  devops: 'DevOps',
  data_engineering: 'Data engineering',
  data_science: 'Data science',
  project_management: 'Project management',
};

const COMPANY_TYPE: Record<string, string> = {
  product: 'Product',
  startup: 'Startup',
  outsource: 'Outsource',
  outstaff: 'Outstaff',
  agency: 'Agency',
  inhouse: 'In-house',
  government: 'Government',
};

const CURRENCY_SYMBOL: Record<string, string> = { USD: '$', EUR: '€', GBP: '£' };

const PERIOD_SUFFIX: Record<string, string> = {
  month: ' / mo',
  day: ' / day',
  hour: ' / hr',
  // `year` is the implicit default and reads cleaner without a suffix.
};

/** Title-case an unknown snake_case code (e.g. "data_engineering" → "Data engineering"). */
function humanize(value: string): string {
  const spaced = value.replace(/_/g, ' ');
  return spaced.charAt(0).toUpperCase() + spaced.slice(1);
}

/** Look a code up in its label map, humanizing anything outside the map. */
function label(map: Record<string, string>, value: string): string {
  return map[value] ?? humanize(value);
}

/** Group thousands with thin spaces, matching the salary line in the design. */
function groupThousands(n: number): string {
  return n.toLocaleString('en-US').replace(/,/g, ' ');
}

/**
 * Render the compensation line, or null when no salary is stated. Handles the
 * full range, a min-only floor ("from …"), and a max-only ceiling ("up to …").
 * The currency symbol and period suffix trail the amount, as in the design.
 */
export function formatSalary(e: Enrichment): string | null {
  const { salary_min, salary_max } = e;
  if (salary_min == null && salary_max == null) return null;

  const symbol = e.salary_currency
    ? (CURRENCY_SYMBOL[e.salary_currency] ?? e.salary_currency)
    : '';
  const period = e.salary_period ? (PERIOD_SUFFIX[e.salary_period] ?? '') : '';
  const tail = `${symbol}${period}`;

  let amount: string;
  if (salary_min != null && salary_max != null) {
    amount = `${groupThousands(salary_min)} – ${groupThousands(salary_max)}`;
  } else if (salary_min != null) {
    amount = `from ${groupThousands(salary_min)}`;
  } else {
    amount = `up to ${groupThousands(salary_max as number)}`;
  }

  return tail ? `${amount} ${tail}` : amount;
}

/**
 * The work-arrangement label for compact contexts (list cards): the enriched
 * `work_mode`, or null when unenriched. "Remote" is purely an enrichment concept
 * now — there is no raw flag to fall back to.
 */
export function workArrangement(job: { enrichment?: Enrichment }): string | null {
  const mode = job.enrichment?.work_mode;
  return mode ? label(WORK_MODE, mode) : null;
}

/**
 * A remote role's geographic reach as a concise label from `regions` — e.g.
 * `Global`, `Europe`, `USA`. Null when the job is not a known remote role or its
 * reach is unknown (empty `regions` is unknown, not global).
 */
export function remoteReach(e: Enrichment | undefined): string | null {
  if (!e || e.work_mode !== 'remote' || !e.regions?.length) return null;
  return e.regions.map((r) => label(REGION, r)).join(', ');
}

/**
 * The short tag row shown on a list card's header: work arrangement, primary
 * country, employment type, and grade — only those that are stated, in that
 * order. Compact by design (the full facet set lives on the detail page).
 */
export function cardTags(job: { enrichment?: Enrichment }): string[] {
  const e = job.enrichment;
  const tags: string[] = [];

  const arrangement = workArrangement(job);
  if (arrangement) tags.push(arrangement);
  const reach = remoteReach(e);
  if (reach) tags.push(reach);
  if (e?.employment_type) tags.push(label(EMPLOYMENT, e.employment_type));
  if (e?.seniority) tags.push(label(SENIORITY, e.seniority));

  return tags;
}

/**
 * Ordered facets for the summary meta-row. Only facets with a stated value are
 * included, so an empty enrichment yields an empty list and the row hides
 * entirely. Order moves from work arrangement → role → eligibility → company,
 * mirroring the reference layout.
 */
export function summaryFacets(e: Enrichment): Facet[] {
  const facets: Facet[] = [];
  const push = (name: string, value: string | null | undefined) => {
    if (value) facets.push({ label: name, value });
  };

  push('Work format', e.work_mode && label(WORK_MODE, e.work_mode));
  push('Reach', remoteReach(e));
  push('Work type', e.employment_type && label(EMPLOYMENT, e.employment_type));
  push('Grade', e.seniority && label(SENIORITY, e.seniority));
  push(
    'Experience',
    e.experience_years_min != null ? `${e.experience_years_min}+ yrs` : null,
  );
  push(
    'English',
    e.english_level && e.english_level !== 'none' ? label(ENGLISH_LEVEL, e.english_level) : null,
  );
  push('Category', e.category && label(CATEGORY, e.category));
  push('Country', e.countries?.length ? e.countries.map((c) => c.toUpperCase()).join(', ') : null);
  push('Relocation', e.relocation && label(RELOCATION, e.relocation));
  push('Visa', e.visa_sponsorship === true ? 'Sponsored' : null);
  push('Company', e.company_type && label(COMPANY_TYPE, e.company_type));
  push('Size', e.company_size);
  push('Domains', e.domains?.length ? e.domains.map((d) => label(DOMAINS, d)).join(', ') : null);

  return facets;
}
