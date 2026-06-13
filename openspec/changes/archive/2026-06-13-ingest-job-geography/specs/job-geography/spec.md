## ADDED Requirements

### Requirement: Job geography is derived deterministically from the location string

The system SHALL provide a deterministic parser that maps a job's free-text
`location` string to a set of ISO 3166-1 alpha-2 country codes and a set of
region codes. The parser SHALL tokenize the location on the separators `,`, `;`,
`/`, `|`, ` - `, and ` or `, and resolve each token against curated dictionaries:
country/city/shorthand names to country codes, macro-region names to region
codes, and country codes to their region. It SHALL emit only values present in
the controlled vocabularies (see below), deduplicated, and SHALL emit nothing for
tokens it cannot resolve (it never guesses). A bare remote marker (e.g. `Remote`)
with no geographic token SHALL yield empty geography; the `global` region SHALL be
emitted only from an explicit open-anywhere marker (e.g. `Anywhere`, `Worldwide`,
`Global`), never inferred from a bare `Remote`.

#### Scenario: A named country yields its code and region

- **WHEN** the location `Remote - Germany` is parsed
- **THEN** the countries are `[de]` and the regions include `eu`

#### Scenario: A macro region name yields a region without a country

- **WHEN** the location `Remote - Europe` is parsed
- **THEN** the regions are `[eu]` and the countries are empty

#### Scenario: Multiple locations union into the result

- **WHEN** the location `Remote - UK or Europe` is parsed
- **THEN** the countries are `[gb]` and the regions include both `uk` and `eu`

#### Scenario: A bare remote marker yields no geography

- **WHEN** the location `Remote` is parsed
- **THEN** both countries and regions are empty

#### Scenario: An explicit open-anywhere marker yields global

- **WHEN** the location `Remote - Anywhere` is parsed
- **THEN** the regions are `[global]`

#### Scenario: An unresolvable location yields no geography

- **WHEN** the location is a token absent from every dictionary
- **THEN** both countries and regions are empty rather than a guessed value

### Requirement: Geography output uses controlled vocabularies

Region codes emitted by the parser SHALL be drawn from the same controlled
vocabulary the enrichment contract defines for `regions` (`global`, the
macro-regions, and the select reach-area country codes), so the parser, the
enrichment contract, and the search facet share one set of values. Country codes
SHALL be ISO 3166-1 alpha-2. A value outside these vocabularies SHALL never be
emitted.

#### Scenario: Parser output validates against the region vocabulary

- **WHEN** any location string is parsed
- **THEN** every emitted region is a member of the controlled region vocabulary
  and every emitted country is a valid ISO 3166-1 alpha-2 code

### Requirement: Job geography is stored on jobs and unioned with enrichment geography at read time

The system SHALL store the parsed geography in two `jobs` columns, `countries`
and `regions` (text arrays, default empty), as source facts derived from the
location — distinct from the AI-derived `enrichment` payload. The public read
model SHALL compute a job's geography as the union of these columns with the
`enrichment.regions`/`enrichment.countries` the enrichment worker produced,
deduplicated. The union SHALL be computed at read time; it SHALL NOT be
materialized into the database, so the ingest and enrichment write paths stay
decoupled.

#### Scenario: Ingest-derived and enrichment-derived geography are unioned

- **WHEN** a job has `regions=[eu]` from its parsed location and
  `enrichment.regions=[emea]` from enrichment, and is read
- **THEN** its geography regions are the deduplicated union `[eu, emea]`

#### Scenario: A job with only parsed geography still reports it

- **WHEN** an unenriched job has `regions=[us]` from its parsed location and is read
- **THEN** its geography regions are `[us]`

### Requirement: The public job object exposes geography as a top-level facet

The public job object SHALL expose geography as top-level `regions` and
`countries` fields carrying the union, reported exactly once. The
`enrichment.regions` and `enrichment.countries` fields SHALL NOT additionally
appear as independent fields in the served object; their values fold into the
top-level union. The stored `enrichment` JSONB SHALL be left untouched (the
enrichment worker's data is preserved).

#### Scenario: Geography appears once, at the top level

- **WHEN** a client reads a job whose enrichment contained `regions`
- **THEN** the returned object carries top-level `regions`/`countries` and does
  not separately repeat `regions`/`countries` under `enrichment`

### Requirement: Existing jobs are backfilled with parsed geography

The system SHALL provide a run-once command that parses the stored `location` of
every existing job and writes the resulting `countries`/`regions`, so geography
is populated for rows that predate this change, including closed jobs that never
re-crawl. The backfill SHALL be idempotent — re-running it converges to the same
result.

#### Scenario: Backfill populates an existing row from its location

- **WHEN** the backfill runs over a job whose `location` is `Remote - USA` and
  whose geography columns are empty
- **THEN** the job's `countries` becomes `[us]` and its `regions` include `us`

#### Scenario: Backfill is idempotent

- **WHEN** the backfill is run twice over the same jobs
- **THEN** the second run produces the same geography as the first
