## MODIFIED Requirements

### Requirement: Enrichment fields follow a typed contract with controlled vocabularies

The system SHALL define the enrichment payload as a single typed Go contract in
`internal/enrich` whose fields and allowed values are the schema's source of
truth. Every field SHALL be optional and omitted when not determined. Enum
fields SHALL accept only their defined vocabulary values; `skills`, `cities`,
`countries`, and `regions` SHALL be arrays; `skills` values SHALL be normalized
lowercase tokens. The contract SHALL provide validation of a payload against the
vocabularies.

The contract SHALL include a remote-scope dimension describing a remote role's
geographic reach: `remote_scope` (enum: `global`, `regional`, `national`) and
`regions` (enum array drawn from a defined region vocabulary: EU, EMEA, EEA, UK,
AMERICAS, NORTH_AMERICA, LATAM, APAC, MENA, AFRICA). An absent `remote_scope`
SHALL mean *unknown* and SHALL be distinct from `global` (open anywhere), which
SHALL be set only on an explicit signal — `global` SHALL NOT be inferred from the
absence of `countries`. `remote_scope` and `regions` are meaningful only when
`work_mode` is `remote`. Validation SHALL check `remote_scope` and each `regions`
element against their vocabularies; it SHALL NOT enforce cross-field consistency
between `remote_scope`, `regions`, and `countries`.

#### Scenario: Payload round-trips through the typed contract

- **WHEN** an `Enrichment` value (e.g. `seniority=senior`, `work_mode=remote`,
  `remote_scope=regional`, `regions=[EU]`, `skills=[go, postgresql]`) is
  marshalled to JSON, stored, read back, and unmarshalled
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

#### Scenario: An out-of-vocabulary remote_scope or region is reported invalid

- **WHEN** the contract validates a payload whose `remote_scope` is `"worldwide"`
  or whose `regions` contains `"europe"` (neither a defined value)
- **THEN** validation reports the payload as invalid, identifying the offending
  field

#### Scenario: Global reach is distinct from unknown reach

- **WHEN** one job's enrichment has `remote_scope=global` with empty `countries`,
  and another's has no `remote_scope` and empty `countries`
- **THEN** the two payloads are distinguishable: the first denotes open-anywhere,
  the second denotes unknown reach

#### Scenario: Cross-field inconsistency is not rejected

- **WHEN** the contract validates a payload whose `remote_scope` is `regional`
  but whose `regions` array is empty
- **THEN** validation does not reject the payload on that basis

### Requirement: The jobs read API exposes enrichment and provenance

The system SHALL include `enrichment`, `enriched_at`, and `enrichment_version` in
the job objects returned by the jobs read endpoints (`GET /api/v1/jobs`,
`GET /api/v1/jobs/:id`, and jobs nested under a company). The public job object
SHALL NOT include the raw `remote` boolean: the public notion of "remote" is
expressed solely through `enrichment.work_mode` (and `enrichment.remote_scope`),
which subsume it. The `jobs.remote` column itself SHALL be retained as an
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
