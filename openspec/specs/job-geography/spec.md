# job-geography Specification

## Purpose
TBD - created by archiving change ingest-job-geography. Update Purpose after archive.
## Requirements
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

The parser SHALL also derive a `work_mode` hint from an explicit marker in the
location string — `remote`, `hybrid`, or `onsite` — checked in priority order
hybrid > remote > onsite (the most specific arrangement wins when several markers
co-occur). A location with no work-mode marker SHALL yield an empty work_mode (a
bare city is never assumed to be onsite). The marker scan is independent of the
geography tokens, so a bare `Remote` yields `work_mode=remote` with empty geography.

#### Scenario: A named country yields its code and region

- **WHEN** the location `Remote - Germany` is parsed
- **THEN** the countries are `[de]` and the regions include `eu`

#### Scenario: A bare remote marker yields a work mode but no geography

- **WHEN** the location `Remote` is parsed
- **THEN** the work_mode is `remote` and both countries and regions are empty

#### Scenario: Work mode marker priority

- **WHEN** a location names both a hybrid and a remote marker (e.g.
  `Hybrid / Remote - London`)
- **THEN** the work_mode is `hybrid`

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
SHALL be ISO 3166-1 alpha-2. The `work_mode` hint SHALL be a member of the
enrichment contract's `work_mode` vocabulary (`remote`, `hybrid`, `onsite`) or
empty. A value outside these vocabularies SHALL never be emitted.

#### Scenario: Parser output validates against the controlled vocabularies

- **WHEN** any location string is parsed
- **THEN** every emitted region is a member of the controlled region vocabulary,
  every emitted country is a valid ISO 3166-1 alpha-2 code, and the work_mode is
  a member of the work-mode vocabulary or empty

### Requirement: Job geography is stored on jobs and unioned with enrichment geography at read time

The system SHALL store the parsed geography in `jobs` columns `countries` and
`regions` (text arrays, default empty) and `work_mode` (text, default empty), as
source facts — distinct from the AI-derived `enrichment` payload. The public read
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

### Requirement: Work mode is resolved by precedence across sources

`work_mode` is a scalar, so it SHALL be resolved by precedence, not union. At
ingest the adapter's STRUCTURED work mode (a workplace-type enum or explicit
remote flag from the ATS) SHALL take precedence over the parser's free-text
heuristic, and the result SHALL be stored in `jobs.work_mode`. At read time the
LLM-derived `enrichment.work_mode` SHALL take precedence over the stored
`jobs.work_mode`, since the LLM reads the whole description. The net order, most
authoritative first, is LLM, then adapter-structured, then parsed location.

#### Scenario: Structured adapter work mode beats the parser

- **WHEN** an adapter reports a structured `work_mode=hybrid` for a posting whose
  location text would parse as `remote`
- **THEN** the stored `jobs.work_mode` is `hybrid`

#### Scenario: The LLM work mode beats the ingest value at read time

- **WHEN** a job has `jobs.work_mode=onsite` from ingest and
  `enrichment.work_mode=remote` from the LLM, and is read
- **THEN** the resolved top-level `work_mode` is `remote`

#### Scenario: The ingest value fills when the LLM did not state work mode

- **WHEN** a job has `jobs.work_mode=hybrid` and no enrichment work_mode, and is read
- **THEN** the resolved top-level `work_mode` is `hybrid`

### Requirement: The public job object exposes geography and work mode as a top-level facet

The public job object SHALL expose geography as top-level `regions` and
`countries` fields carrying the union, and `work_mode` as a top-level field
carrying the resolved value, each reported exactly once. The
`enrichment.regions`, `enrichment.countries`, and `enrichment.work_mode` fields
SHALL NOT additionally appear as independent fields in the served object; their
values fold into the top-level facet. The stored `enrichment` JSONB SHALL be left
untouched (the enrichment worker's data is preserved).

#### Scenario: Geography and work mode appear once, at the top level

- **WHEN** a client reads a job whose enrichment contained `regions` and `work_mode`
- **THEN** the returned object carries top-level `regions`/`countries`/`work_mode`
  and does not separately repeat those fields under `enrichment`

### Requirement: Existing jobs are backfilled with parsed geography

The system SHALL provide a run-once command that parses the stored `location` of
every existing job and writes the resulting `countries`/`regions`/`work_mode`, so
the location-derived facets are populated for rows that predate this change,
including closed jobs that never re-crawl. The backfill SHALL be idempotent —
re-running it converges to the same result. Because the original structured ATS
signal is not available at backfill time, the backfill SHALL fill `work_mode` from
the parsed location only when the row's `work_mode` is empty, preserving any value
already set (a later re-crawl refreshes it with the structured value).

#### Scenario: Backfill populates an existing row from its location

- **WHEN** the backfill runs over a job whose `location` is `Remote - USA` and
  whose geography columns are empty
- **THEN** the job's `countries` becomes `[us]` and its `regions` include `us`

#### Scenario: Backfill is idempotent

- **WHEN** the backfill is run twice over the same jobs
- **THEN** the second run produces the same geography as the first

