## ADDED Requirements

### Requirement: Orphan jobs are liveness-probed by URL

The system SHALL probe the posting URL of every open job whose `source` is not a
registered ATS board provider — the sources no ingest run re-crawls (e.g.
`telegram`, `habr_career`, `geekjob`). Board-provider jobs, which the ingest sweep
already covers, SHALL NOT be probed. The probe SHALL use a plain HTTP request (no
headless browser, no LLM) with a per-probe timeout, and SHALL classify the outcome
without any persisted page content.

#### Scenario: Orphan job is a probe candidate

- **WHEN** the liveness worker runs and an open job has `source = 'telegram'`
- **THEN** that job's posting URL is fetched and classified

#### Scenario: Board job is not probed

- **WHEN** the liveness worker runs and an open job has `source = 'greenhouse'`
  (a registered ATS provider)
- **THEN** that job is not selected for probing

#### Scenario: Closed job is not probed

- **WHEN** the liveness worker runs and an orphan job already has `closed_at` set
- **THEN** that job is not selected for probing

### Requirement: A probe is classified expired only on a definitive death signal

The classifier SHALL return `expired` only when the fetch yields a definitive signal
that the posting is gone: an HTTP `404` or `410`; a final URL matching an
error/listing redirect pattern; a response body matching a curated hard-expired
pattern; or body content below a minimum length threshold. Any other outcome —
including `5xx`, `403`, a network or timeout error, healthy content, or a
client-rendered shell with no server-side closed message — SHALL be classified as
not-expired and SHALL trigger no state change.

#### Scenario: HTTP gone is expired

- **WHEN** a probe returns HTTP 404 or 410
- **THEN** the probe is classified `expired`

#### Scenario: Closed-posting body is expired

- **WHEN** a probe returns HTTP 200 with a body matching a hard-expired pattern
  (e.g. "no longer accepting applications")
- **THEN** the probe is classified `expired`

#### Scenario: Empty shell is expired

- **WHEN** a probe returns body content shorter than the minimum content threshold
- **THEN** the probe is classified `expired`

#### Scenario: Transient failure is not expired

- **WHEN** a probe returns HTTP 503, 403, or fails with a timeout
- **THEN** the probe is classified not-expired and no state change is made

#### Scenario: Healthy page is not expired

- **WHEN** a probe returns HTTP 200 with substantial content and no hard-expired
  signal
- **THEN** the probe is classified not-expired and no state change is made

### Requirement: An orphan job is closed only after two consecutive expired probes

The system SHALL track consecutive `expired` probes per job in
`jobs.liveness_strikes`. An `expired` probe SHALL increment the counter, and on
reaching two SHALL set `closed_at` within the same write. Any not-expired probe
SHALL reset the counter to zero, so two non-consecutive expired reads never close a
job. This grace absorbs a transient death signal and biases toward leaving an orphan
job open rather than closing it irreversibly.

#### Scenario: First expired probe stamps a strike but does not close

- **WHEN** an open orphan job with `liveness_strikes = 0` is probed `expired`
- **THEN** `liveness_strikes` becomes 1 and `closed_at` remains NULL

#### Scenario: Second consecutive expired probe closes the job

- **WHEN** an open orphan job with `liveness_strikes = 1` is probed `expired`
- **THEN** `liveness_strikes` becomes 2 and `closed_at` is set, and the job stops
  appearing in list and search surfaces

#### Scenario: A healthy probe resets the strike count

- **WHEN** an open orphan job with `liveness_strikes = 1` is probed not-expired
- **THEN** `liveness_strikes` is reset to 0 and the job remains open
