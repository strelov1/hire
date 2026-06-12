## MODIFIED Requirements

### Requirement: Searchable jobs index

The system SHALL maintain a Meilisearch index of jobs with one document per job,
keyed by the job's internal `id`. Each document SHALL carry the fields needed to
both match and render a result without a follow-up database read: the searchable
text (title, company, description, location), the filterable facets, the
sortable fields, and the display fields returned to clients.

The index SHALL declare:
- **searchable attributes**: title, company, description, location.
- **filterable attributes**: source, company_slug, work_mode, employment_type,
  seniority, category, domains, countries, company_type, company_size,
  visa_sponsorship, salary_currency, salary_period, skills, salary_min,
  salary_max, experience_years_min, and the derived `remote_type`. The raw
  `remote` flag SHALL NOT be a filterable attribute (work_mode subsumes it).
- **sortable attributes**: posted_at, salary_min, salary_max.

The document SHALL carry a derived, multi-valued `remote_type` field computed
from the enrichment and gated on `work_mode = remote`: `remote_scope = global`
yields `["global"]`; `regional` yields the lowercased `regions`; `national`
yields the lowercased `countries`; an unknown scope or a non-remote work mode
yields no values. `remote_type` SHALL exist only on the search document (it is a
facet denormalization) and SHALL NOT be added to the enrichment contract or the
non-search read API.

Facets derived from a job's `enrichment` JSONB SHALL be absent (or empty) on the
document when the job is not yet enriched; an unenriched job SHALL still be
indexed and findable by its text fields.

#### Scenario: A job is represented as one searchable document

- **WHEN** a job with title "Senior Go Developer", company "Acme", and a
  description is indexed
- **THEN** the `jobs` index holds one document keyed by that job's `id` whose
  searchable text includes the title, company, and description

#### Scenario: Unenriched job is still indexed

- **WHEN** a job with no enrichment is indexed
- **THEN** the document is present and matchable by its title/company/description
  text, with its enrichment-derived facets absent or empty

#### Scenario: Remote type is derived for a remote job

- **WHEN** a job with `work_mode=remote` and `remote_scope=national`,
  `countries=[US]` is indexed
- **THEN** its document's `remote_type` includes `us`

#### Scenario: Global reach is derived distinctly from unknown

- **WHEN** a job with `work_mode=remote` and `remote_scope=global` is indexed,
  and another remote job has no `remote_scope`
- **THEN** the first document's `remote_type` is `["global"]` and the second
  document's `remote_type` is absent/empty

### Requirement: Public job search endpoint

The system SHALL expose `GET /api/v1/jobs/search` as a public (unauthenticated)
endpoint. It SHALL accept a free-text query `q`, facet filters matching the
index's filterable attributes, an optional sort, an optional semantic ratio, and
`limit`/`offset` pagination. Facet filters SHALL include `remote_type` (matching
the derived field) and SHALL NOT include the removed raw `remote` filter. The
response SHALL use the standard list envelope `{"data": [...], "meta": {...}}`,
where `data` is the matched job documents and `meta` carries at least the
estimated total hit count and the applied `limit`/`offset`. The existing
`GET /api/v1/jobs` list endpoint SHALL be unchanged.

Each result SHALL identify its job by `public_slug` and SHALL NOT include the
internal numeric `id`, consistent with the public-identity contract used by the
other public job reads.

#### Scenario: Keyword query returns matches

- **WHEN** a client requests `GET /api/v1/jobs/search?q=golang`
- **THEN** the response is `{"data": [...], "meta": {...}}` with jobs matching
  "golang" in `data` and the estimated total and pagination in `meta`

#### Scenario: Faceted filtering by remote type

- **WHEN** a client requests
  `GET /api/v1/jobs/search?q=engineer&seniority=senior&remote_type=eu`
- **THEN** only jobs whose facets satisfy seniority=senior AND whose
  `remote_type` includes `eu` are returned

#### Scenario: Empty query browses with filters

- **WHEN** a client requests `GET /api/v1/jobs/search` with filters but no `q`
- **THEN** the filtered jobs are returned ranked by the index defaults

#### Scenario: Pagination is reflected in meta

- **WHEN** a client requests `GET /api/v1/jobs/search?q=go&limit=10&offset=20`
- **THEN** at most 10 documents are returned and `meta` reports the applied
  `limit` 10 and `offset` 20 alongside the estimated total

#### Scenario: Results identify jobs by public slug, not internal id

- **WHEN** a job is returned by `GET /api/v1/jobs/search`
- **THEN** the result carries the job's `public_slug` and omits the internal
  numeric `id`
