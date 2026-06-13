## ADDED Requirements

### Requirement: Ingest persists job geography parsed from the location

The ingest write path SHALL parse each posting's `location` string into
`countries`/`regions` (via the job-geography parser) and persist them on the job
row. These columns SHALL be written on insert and refreshed on re-ingest, like
the other raw source fields and unlike the enrichment payload (which ingest never
writes). A posting whose location yields no geography SHALL store empty arrays.

#### Scenario: A new posting stores its parsed geography

- **WHEN** a posting with location `Remote - Germany` is ingested
- **THEN** the stored job has `countries=[de]` and `regions` including `eu`

#### Scenario: Re-ingest refreshes geography from the updated location

- **WHEN** an already-ingested posting is re-ingested with its location changed
  from `Remote - UK` to `Remote - USA`
- **THEN** the job's `countries` updates to `[us]` and its `regions` update
  accordingly

#### Scenario: A location with no geography stores empty arrays

- **WHEN** a posting with location `Remote` is ingested
- **THEN** the stored job has empty `countries` and empty `regions`
