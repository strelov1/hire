// The job-search facets, as a data-driven registry. Each entry declares its
// query param (matching the backend search API), section label, the control to
// render, its options (for pills/select), and whether it supports an "exclude"
// mode. The panel iterates this list; the store keys facet state by `param`.
//
// SOURCE OF TRUTH for the closed-vocabulary option values below (work mode,
// seniority, category, employment type, domains, relocation, English level,
// company type/size, salary period): internal/enrich/enrichment.go, which the
// enrichment worker validates against. When a vocabulary changes there, update
// the matching list here. Drift is not fatal — enrichment.ts's humanize()
// renders an unknown value as a readable label rather than blank — but the facet
// would silently stop offering it as a filter.

export interface FacetOption {
  value: string;
  label: string;
}

export type FacetControl = 'pills' | 'select' | 'tokens';

export interface FacetDef {
  param: string;
  label: string;
  control: FacetControl;
  options?: FacetOption[];
  excludable: boolean;
  /** Show a per-facet AND/OR toggle (match all vs match any) over selected values. */
  hasAndOr?: boolean;
  placeholder?: string;
}

const WORK_MODE: FacetOption[] = [
  { value: 'remote', label: 'Remote' },
  { value: 'hybrid', label: 'Hybrid' },
  { value: 'onsite', label: 'On-site' },
];

// The platform a job was ingested from. Unlike the enrichment facets below, this
// value is always present (the ingest pipeline sets it), so it is the one fully
// reliable filter. SOURCE OF TRUTH for these values: the Provider() strings of the
// adapters in internal/sources (sources.All) plus the literal "telegram" set by
// the tg-extract worker. A new adapter means a new entry here.
const SOURCE: FacetOption[] = [
  { value: 'telegram', label: 'Telegram' },
  { value: 'greenhouse', label: 'Greenhouse' },
  { value: 'lever', label: 'Lever' },
  { value: 'ashby', label: 'Ashby' },
  { value: 'workable', label: 'Workable' },
  { value: 'recruitee', label: 'Recruitee' },
  { value: 'smartrecruiters', label: 'SmartRecruiters' },
  { value: 'personio', label: 'Personio' },
  { value: 'pinpoint', label: 'Pinpoint' },
  { value: 'rippling', label: 'Rippling' },
  { value: 'bamboohr', label: 'BambooHR' },
  { value: 'workday', label: 'Workday' },
];

// A curated, extensible subset of the backend's `regions` reach vocabulary. Its
// values mix levels by design (global / region / country); the field's full
// vocabulary holds more than these pills surface.
const REGION: FacetOption[] = [
  { value: 'global', label: 'Global' },
  { value: 'ru', label: 'Russia' },
  { value: 'eu', label: 'Europe' },
  { value: 'us', label: 'USA' },
];

const SENIORITY: FacetOption[] = [
  { value: 'intern', label: 'Intern' },
  { value: 'junior', label: 'Junior' },
  { value: 'middle', label: 'Middle' },
  { value: 'senior', label: 'Senior' },
  { value: 'lead', label: 'Lead' },
  { value: 'principal', label: 'Principal' },
  { value: 'c_level', label: 'C-level' },
];

const COMPANY_TYPE: FacetOption[] = [
  { value: 'startup', label: 'Startup' },
  { value: 'product', label: 'Product' },
  { value: 'outsource', label: 'Outsource' },
  { value: 'outstaff', label: 'Outstaff' },
  { value: 'agency', label: 'Agency' },
  { value: 'inhouse', label: 'In-house' },
  { value: 'government', label: 'Government' },
];

const EMPLOYMENT: FacetOption[] = [
  { value: 'full_time', label: 'Full-time' },
  { value: 'part_time', label: 'Part-time' },
  { value: 'contract', label: 'Contract' },
  { value: 'internship', label: 'Internship' },
];

const RELOCATION: FacetOption[] = [
  { value: 'not_supported', label: 'None' },
  { value: 'supported', label: 'Supported' },
  { value: 'required', label: 'Required' },
];

const ENGLISH: FacetOption[] = [
  { value: 'a1', label: 'A1' },
  { value: 'a2', label: 'A2' },
  { value: 'b1', label: 'B1' },
  { value: 'b2', label: 'B2' },
  { value: 'c1', label: 'C1' },
  { value: 'c2', label: 'C2' },
  { value: 'native', label: 'Native' },
];

const POSTING_LANGUAGE: FacetOption[] = [
  { value: 'en', label: 'EN' },
  { value: 'ru', label: 'RU' },
  { value: 'uk', label: 'UA' },
];

const CURRENCY: FacetOption[] = [
  { value: 'USD', label: 'USD' },
  { value: 'EUR', label: 'EUR' },
  { value: 'GBP', label: 'GBP' },
  { value: 'RUB', label: 'RUB' },
];

const CATEGORY: FacetOption[] = [
  { value: 'backend', label: 'Backend' },
  { value: 'frontend', label: 'Frontend' },
  { value: 'fullstack', label: 'Fullstack' },
  { value: 'mobile', label: 'Mobile' },
  { value: 'devops', label: 'DevOps' },
  { value: 'data_engineering', label: 'Data Engineering' },
  { value: 'data_science', label: 'Data Science' },
  { value: 'ml_ai', label: 'ML / AI' },
  { value: 'qa', label: 'QA' },
  { value: 'security', label: 'Security' },
  { value: 'design', label: 'Design' },
  { value: 'product', label: 'Product' },
  { value: 'project_management', label: 'Project Management' },
  { value: 'management', label: 'Management' },
  { value: 'marketing', label: 'Marketing' },
  { value: 'sales', label: 'Sales' },
  { value: 'support', label: 'Support' },
  { value: 'other', label: 'Other' },
];

const DOMAINS: FacetOption[] = [
  { value: 'fintech', label: 'Fintech' },
  { value: 'gambling', label: 'Gambling' },
  { value: 'ecommerce', label: 'E-commerce' },
  { value: 'crypto', label: 'Crypto' },
  { value: 'healthcare', label: 'Healthcare' },
  { value: 'saas', label: 'SaaS' },
  { value: 'gamedev', label: 'Gamedev' },
  { value: 'edtech', label: 'Edtech' },
  { value: 'adtech', label: 'Adtech' },
  { value: 'govtech', label: 'Govtech' },
  { value: 'media', label: 'Media' },
  { value: 'travel', label: 'Travel' },
  { value: 'logistics', label: 'Logistics' },
  { value: 'other', label: 'Other' },
];

export const FACETS: FacetDef[] = [
  { param: 'source', label: 'Source', control: 'pills', options: SOURCE, excludable: true },
  { param: 'work_mode', label: 'Work format', control: 'pills', options: WORK_MODE, excludable: true },
  { param: 'regions', label: 'Region', control: 'pills', options: REGION, excludable: true },
  { param: 'seniority', label: 'Seniority', control: 'pills', options: SENIORITY, excludable: true },
  { param: 'category', label: 'Specialization', control: 'select', options: CATEGORY, excludable: true, placeholder: 'Search specializations' },
  { param: 'skills', label: 'Skills', control: 'tokens', excludable: true, hasAndOr: true, placeholder: 'Add a skill, press Enter' },
  { param: 'domains', label: 'Industry', control: 'select', options: DOMAINS, excludable: true, placeholder: 'Search industries' },
  { param: 'company_type', label: 'Company type', control: 'pills', options: COMPANY_TYPE, excludable: true },
  { param: 'countries', label: 'Countries', control: 'tokens', excludable: true, placeholder: 'ISO code, e.g. DE' },
  { param: 'relocation', label: 'Relocation', control: 'pills', options: RELOCATION, excludable: true },
  { param: 'employment_type', label: 'Employment', control: 'pills', options: EMPLOYMENT, excludable: true },
  { param: 'english_level', label: 'English', control: 'pills', options: ENGLISH, excludable: true },
  { param: 'posting_language', label: 'Job language', control: 'pills', options: POSTING_LANGUAGE, excludable: true },
  { param: 'salary_currency', label: 'Currency', control: 'pills', options: CURRENCY, excludable: true },
];
