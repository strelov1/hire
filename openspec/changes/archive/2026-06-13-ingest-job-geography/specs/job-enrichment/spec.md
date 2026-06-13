## MODIFIED Requirements

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
each `regions` element against the vocabulary. The enrichment-derived `regions`
and `countries` are an *additive* source: at read time they fold into the
top-level job geography union (see the job-geography capability) rather than being
served as independent enrichment fields.

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

### Requirement: The jobs read API exposes enrichment and provenance

The system SHALL include `enrichment`, `enriched_at`, and `enrichment_version` in
the job objects returned by the jobs read endpoints (`GET /api/v1/jobs`,
`GET /api/v1/jobs/:id`, and jobs nested under a company). The public job object
SHALL expose geography as top-level `regions` and `countries` fields (the union of
the parsed-location columns and the enrichment-derived values); these geography
fields SHALL NOT additionally appear as independent fields under `enrichment`. The
public job object SHALL NOT include the raw `remote` boolean: the public notion of
"remote" is expressed solely through `enrichment.work_mode` (and the top-level
geography for area), which subsume it. The `jobs.remote` column itself SHALL be
retained as an internal enrichment input and SHALL NOT be removed.

#### Scenario: Job detail includes enrichment and provenance

- **WHEN** a client requests `GET /api/v1/jobs/:id` for an existing job
- **THEN** the returned object under `data` includes `enrichment`,
  `enriched_at`, and `enrichment_version` alongside the existing fields

#### Scenario: Empty enrichment serializes as an object

- **WHEN** a job that has not been enriched is returned by a read endpoint
- **THEN** its `enrichment` is serialized as an empty object (`{}`), not null

#### Scenario: Geography is served top-level, not duplicated under enrichment

- **WHEN** a client reads a job whose enrichment contained `regions`/`countries`
- **THEN** the returned object carries top-level `regions`/`countries` and its
  `enrichment` object does not separately repeat those fields

#### Scenario: The raw remote flag is absent from the public job object

- **WHEN** a client requests any jobs read endpoint
- **THEN** the returned job objects do not contain a top-level `remote` field
