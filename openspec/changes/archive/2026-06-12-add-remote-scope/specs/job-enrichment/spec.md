## MODIFIED Requirements

### Requirement: Enrichment fields follow a typed contract with controlled vocabularies

The system SHALL define the enrichment payload as a single typed Go contract in
`internal/enrich` whose fields and allowed values are the schema's source of
truth. Every field SHALL be optional and omitted when not determined. Enum
fields SHALL accept only their defined vocabulary values; `skills`, `cities`,
`countries`, and `regions` SHALL be arrays; `skills` values SHALL be normalized
lowercase tokens. The contract SHALL provide validation of a payload against the
vocabularies.

The contract SHALL capture a remote role's geographic reach in a single `regions`
field — an enum array of reach codes drawn from one controlled vocabulary that
mixes levels: `global` (open anywhere), macro-regions (`eu`, `emea`, `eea`, `uk`,
`americas`, `north_america`, `latam`, `apac`, `mena`, `africa`), and select
countries treated as reach areas (e.g. `us`, `ru`). There SHALL be no separate
remote-scope discriminator field: an absent/empty `regions` means *unknown*, and
`global` is an explicit value (never inferred from the absence of other reach
codes), so open-anywhere is distinct from unknown. `regions` is meaningful only
when `work_mode` is `remote`. Validation SHALL check each `regions` element
against the vocabulary.

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
  the second denotes unknown reach

### Requirement: The jobs read API exposes enrichment and provenance

The system SHALL include `enrichment`, `enriched_at`, and `enrichment_version` in
the job objects returned by the jobs read endpoints (`GET /api/v1/jobs`,
`GET /api/v1/jobs/:id`, and jobs nested under a company). The public job object
SHALL NOT include the raw `remote` boolean: the public notion of "remote" is
expressed solely through `enrichment.work_mode` (and `enrichment.regions` for
reach), which subsume it. The `jobs.remote` column itself SHALL be retained as an
internal enrichment input and SHALL NOT be removed.

#### Scenario: Job detail includes enrichment and provenance

- **WHEN** a client requests `GET /api/v1/jobs/:id` for an existing job
- **THEN** the returned object under `data` includes `enrichment`,
  `enriched_at`, and `enrichment_version` alongside the existing fields

#### Scenario: Empty enrichment serializes as an object

- **WHEN** a job that has not been enriched is returned by a read endpoint
- **THEN** its `enrichment` is serialized as an empty object (`{}`), not null

#### Scenario: The raw remote flag is absent from the public job object

- **WHEN** a client requests any jobs read endpoint
- **THEN** the returned job objects do not contain a top-level `remote` field
