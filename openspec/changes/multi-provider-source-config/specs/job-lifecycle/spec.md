## MODIFIED Requirements

### Requirement: Jobs unseen beyond a grace window are closed after a run

After an ingest run, the system SHALL run the unseen-job sweep **per provider**: for each
provider that ingested at least one job during the run, it SHALL stamp `closed_at` on every
open job of that provider whose `last_seen_at` is older than a 48-hour grace window. A
provider that ingested nothing in the run SHALL NOT have its jobs swept, so a total crawl
failure — for one provider in a multi-provider run, or for a whole single-provider run —
cannot mass-close that provider's catalogue. The sweep of one provider never touches
another provider's jobs.

#### Scenario: Stale job is closed

- **WHEN** a sweep runs after a provider ingested at least one job and an open job of that
  provider was last seen 49 hours ago
- **THEN** that job's `closed_at` is set and the job stops appearing in list surfaces

#### Scenario: Recently seen job survives the sweep

- **WHEN** a sweep runs and an open job was last seen 6 hours ago
- **THEN** the job remains open

#### Scenario: A provider that ingested nothing closes nothing

- **WHEN** a run ingested jobs for provider A but zero for provider B (B's crawl failed)
- **THEN** the sweep runs for A but not for B, so no B job is closed

#### Scenario: One provider's sweep leaves another provider's jobs alone

- **WHEN** a multi-provider run sweeps provider A's stale jobs
- **THEN** provider B's jobs are never closed by A's sweep
