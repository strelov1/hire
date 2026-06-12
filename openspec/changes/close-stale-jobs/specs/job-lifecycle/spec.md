## ADDED Requirements

### Requirement: Every ingested job records when a crawl last saw it

The system SHALL stamp `last_seen_at` on a job every time ingest upserts it, for
both newly inserted and re-ingested postings, within the same atomic write that
persists the job.

#### Scenario: Re-ingest refreshes liveness

- **WHEN** an ingest run upserts a job that already exists
- **THEN** the job's `last_seen_at` is set to the time of that write

### Requirement: Jobs unseen beyond a grace window are closed after a run

After an ingest run that ingested at least one job, the system SHALL stamp
`closed_at` on every open job whose `last_seen_at` is older than a 48-hour grace
window. A run that ingested nothing SHALL NOT run the sweep, so a total crawl
failure cannot mass-close the catalogue.

#### Scenario: Stale job is closed

- **WHEN** a sweep runs after a successful ingest and an open job was last seen 49 hours ago
- **THEN** that job's `closed_at` is set and the job stops appearing in list surfaces

#### Scenario: Recently seen job survives the sweep

- **WHEN** a sweep runs and an open job was last seen 6 hours ago
- **THEN** the job remains open

#### Scenario: Failed run closes nothing

- **WHEN** an ingest run ends with zero ingested jobs
- **THEN** the sweep does not run and no job is closed

### Requirement: A reappearing posting reopens its job

The system SHALL clear `closed_at` when ingest upserts a job that was previously
closed, restoring it to all open-job surfaces.

#### Scenario: Republished posting reopens

- **WHEN** a closed job's posting appears again in a crawl
- **THEN** the upsert clears `closed_at` and the job is listed again

### Requirement: Closed jobs are hidden from lists but served on detail

The jobs list SHALL return only open jobs. The job detail endpoint SHALL still
return a closed job — its public slug, enrichment, and a `closed_at` timestamp in
the job view shape — so external links and application history never break.

#### Scenario: Closed job leaves the list

- **WHEN** a job has `closed_at` set
- **THEN** `GET /api/v1/jobs` does not include it

#### Scenario: Closed job detail still resolves

- **WHEN** a client requests `GET /api/v1/jobs/:slug` for a closed job
- **THEN** the response is 200 and the job view carries its `closed_at`
