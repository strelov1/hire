# job-enrichment

## Purpose

Capture AI-derived, additive metadata for each job — seniority, work mode,
skills, salary, location descriptors, and company descriptors — in a typed,
versioned enrichment payload, so jobs can be filtered and presented richly
without altering the raw source fields ingested by parsers.
## Requirements
### Requirement: Jobs store an additive enrichment payload

The system SHALL store a structured enrichment payload per job in a
`jobs.enrichment` JSONB column that defaults to an empty object. Enrichment SHALL
be additive: writing it SHALL NOT modify any raw source field (`title`,
`company`, `location`, `remote`, `description`, `posted_at`, `company_slug`).

#### Scenario: New job defaults to an empty payload

- **WHEN** a job is upserted without an enrichment payload
- **THEN** its `enrichment` reads back as an empty object (`{}`) and its raw
  fields are stored unchanged

#### Scenario: Enrichment is stored without altering raw fields

- **WHEN** a job is upserted with an enrichment payload
- **THEN** the payload is persisted under `enrichment` and the job's raw fields
  remain exactly as supplied

### Requirement: Enrichment fields follow a typed contract with controlled vocabularies

The system SHALL define the enrichment payload as a single typed Go contract in
`internal/enrich` whose fields and allowed values are the schema's source of
truth. Every field SHALL be optional and omitted when not determined. Enum
fields SHALL accept only their defined vocabulary values; `skills`, `cities`,
`countries`, and `regions` SHALL be arrays; `skills` values SHALL be normalized
lowercase tokens. The contract SHALL provide validation of a payload against the
vocabularies.

The contract SHALL capture a job's geographic area in a single `regions` field —
an enum array of codes drawn from one controlled vocabulary that mixes levels:
`global` (open anywhere), macro-regions (`eu`, `emea`, `eea`, `uk`, `americas`,
`north_america`, `latam`, `apac`, `mena`, `africa`), and select countries treated
as area codes (e.g. `us`, `ru`). `regions` denotes the geographic area of the job
and is meaningful for any `work_mode` (for a remote role its reach, for an onsite
role its office area); the prior restriction to remote roles is removed. There
SHALL be no separate scope discriminator field: an absent/empty `regions` means
*unknown*, and `global` is an explicit value (never inferred from the absence of
other codes), so open-anywhere is distinct from unknown. Validation SHALL check
each `regions` element against the vocabulary. The enrichment-derived `regions`,
`countries`, and `work_mode` are an *additive* source: at read time they fold into
the top-level job geography facet (see the job-geography capability) — geography by
union, `work_mode` by precedence (the LLM value winning over the ingest one) —
rather than being served as independent enrichment fields.

#### Scenario: Payload round-trips through the typed contract

- **WHEN** an `Enrichment` value (e.g. `seniority=senior`, `work_mode=remote`,
  `regions=[eu]`, `skills=[go, postgresql]`) is marshalled to JSON, stored, read
  back, and unmarshalled
- **THEN** the resulting value equals the original

#### Scenario: Undetermined fields are omitted, not zero-filled

- **WHEN** an enrichment payload does not determine salary
- **THEN** the `salary_min`, `salary_max`, `salary_currency`, and
  `salary_period` keys are absent from the stored JSON rather than present with
  zero/empty values

#### Scenario: A value outside a vocabulary is reported invalid

- **WHEN** the contract validates a payload whose `seniority` is `"sr"` (not a
  defined value)
- **THEN** validation reports the payload as invalid, identifying the offending
  field

#### Scenario: An out-of-vocabulary region is reported invalid

- **WHEN** the contract validates a payload whose `regions` contains `"europe"`
  (not a defined value)
- **THEN** validation reports the payload as invalid, identifying the offending
  `regions` field

#### Scenario: Global reach is distinct from unknown reach

- **WHEN** one job's enrichment has `regions=[global]`, and another's has empty
  `regions`
- **THEN** the two payloads are distinguishable: the first denotes open-anywhere,
  the second denotes unknown

### Requirement: Enrichment provenance is tracked per job

The system SHALL track enrichment provenance with two job columns: `enriched_at`
(nullable timestamp) and `enrichment_version` (integer, default 0). A job that
has never been enriched SHALL have `enriched_at` null and `enrichment_version` 0,
so un-enriched rows are identifiable for later processing.

#### Scenario: Un-enriched job is identifiable

- **WHEN** a job has never been enriched
- **THEN** its `enriched_at` is null and its `enrichment_version` is 0

#### Scenario: Provenance reflects a completed enrichment

- **WHEN** a job is written with an enrichment payload produced at schema
  version N
- **THEN** its `enriched_at` is set and its `enrichment_version` equals N

### Requirement: Company descriptors are captured as job enrichment fields

The system SHALL capture company descriptors (`company_type`, `company_size`) as
fields of the job's enrichment payload, not as columns on the `companies` table.
Writing them SHALL NOT alter any `companies` row.

#### Scenario: Company descriptors live in the job payload

- **WHEN** a job is upserted with enrichment including `company_type=product`
- **THEN** the value is stored in that job's `enrichment` and no `companies` row
  is created or modified by it

### Requirement: The jobs read API exposes enrichment and provenance

The system SHALL include `enrichment`, `enriched_at`, and `enrichment_version` in
the job objects returned by the jobs read endpoints (`GET /api/v1/jobs`,
`GET /api/v1/jobs/:id`, and jobs nested under a company). The public job object
SHALL expose geography as top-level `regions` and `countries` fields (the union of
the parsed-location columns and the enrichment-derived values) and `work_mode` as
a top-level field (the LLM value when present, else the ingest-derived one); these
fields SHALL NOT additionally appear as independent fields under `enrichment`. The
public job object SHALL NOT include the raw `remote` boolean: the public notion of
"remote" is expressed solely through the top-level `work_mode` (and the top-level
geography for area), which subsume it. The `jobs.remote` column itself SHALL be
retained as an internal enrichment input and SHALL NOT be removed.

#### Scenario: Job detail includes enrichment and provenance

- **WHEN** a client requests `GET /api/v1/jobs/:id` for an existing job
- **THEN** the returned object under `data` includes `enrichment`,
  `enriched_at`, and `enrichment_version` alongside the existing fields

#### Scenario: Empty enrichment serializes as an object

- **WHEN** a job that has not been enriched is returned by a read endpoint
- **THEN** its `enrichment` is serialized as an empty object (`{}`), not null

#### Scenario: Geography and work mode are served top-level, not duplicated under enrichment

- **WHEN** a client reads a job whose enrichment contained
  `regions`/`countries`/`work_mode`
- **THEN** the returned object carries top-level `regions`/`countries`/`work_mode`
  and its `enrichment` object does not separately repeat those fields

#### Scenario: The raw remote flag is absent from the public job object

- **WHEN** a client requests any jobs read endpoint
- **THEN** the returned job objects do not contain a top-level `remote` field

