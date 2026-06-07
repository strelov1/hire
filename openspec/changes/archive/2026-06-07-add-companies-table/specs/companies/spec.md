## ADDED Requirements

### Requirement: Companies are stored as a slug-keyed entity

The system SHALL store companies in a `companies` table identified by a natural
`slug` key derived by normalizing the company name. The table SHALL NOT use a
surrogate id. Each company SHALL have a display `name`.

#### Scenario: Company is created from a job's company name

- **WHEN** a job is ingested with a non-empty company name that has no matching
  company row
- **THEN** the system inserts a `companies` row whose `slug` is the normalized
  name and whose `name` is the display name

#### Scenario: Existing company is reused, not duplicated

- **WHEN** a job is ingested whose normalized company name matches an existing
  `companies.slug`
- **THEN** no duplicate company row is created and the existing row is reused

### Requirement: Jobs link to a company via a denormalized key

The system SHALL store `company_slug` on each job as the normalized link key,
kept alongside the existing `company` display name. Jobs with an empty company
name SHALL have an empty `company_slug` and SHALL NOT create a company.

#### Scenario: Job carries both display name and link key

- **WHEN** a job with company name "Yandex LLC" is ingested
- **THEN** the job's `company` is the display name and its `company_slug` is the
  normalized key, and a matching `companies` row exists with that `slug`

#### Scenario: Job with no company

- **WHEN** a job is ingested with an empty company name
- **THEN** the job is stored with an empty `company_slug` and no company row is
  created

### Requirement: Company list is served without joining jobs

The system SHALL expose `GET /api/v1/companies` returning companies read from the
`companies` table. Job counts, when included, SHALL be computed at query time;
no denormalized counter is required.

#### Scenario: Listing companies

- **WHEN** a client requests `GET /api/v1/companies`
- **THEN** the response contains companies under `data` with list `meta`,
  following the existing list response shape

### Requirement: Company detail returns the company with its jobs

The system SHALL expose `GET /api/v1/companies/:slug` returning the company and
its jobs. The company SHALL be read from `companies` and its jobs from a
single-table filter on `jobs.company_slug` — without a SQL join between the two
tables.

#### Scenario: Existing company

- **WHEN** a client requests `GET /api/v1/companies/:slug` for an existing slug
- **THEN** the response contains the company and its jobs ordered like the main
  jobs listing

#### Scenario: Unknown company

- **WHEN** a client requests `GET /api/v1/companies/:slug` for a slug with no
  company row
- **THEN** the system responds with HTTP 404
