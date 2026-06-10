## ADDED Requirements

### Requirement: Jobs needing enrichment are tracked in a durable outbox queue

The system SHALL maintain an `enrichment_outbox` table holding one entry per
`(job_id, target_version)` that needs enriching. The entry SHALL reference the job by
id and SHALL NOT duplicate the job's source fields. The system SHALL provide an
idempotent enqueue that adds entries for jobs whose `enriched_at IS NULL` or whose
`enrichment_version` is below the current schema version (`enrich.Version`);
re-enqueuing an already-queued `(job_id, target_version)` SHALL NOT create a duplicate.

#### Scenario: Pending jobs are enqueued

- **WHEN** the enqueue runs and a job has `enriched_at = NULL`
- **THEN** an outbox entry for that job at the current `target_version` exists

#### Scenario: Stale-version jobs are enqueued

- **WHEN** a job's `enrichment_version` is below the current `enrich.Version`
- **THEN** an outbox entry for that job at the current `target_version` exists

#### Scenario: Enqueue is idempotent

- **WHEN** the enqueue runs twice without the job being enriched in between
- **THEN** the job has exactly one outbox entry for that `target_version`

### Requirement: Enrichment is extracted from a job's description by an LLM provider

The system SHALL define a `Provider` abstraction in `internal/enrich` that, given a
job's source fields (at minimum `title`, `company`, `location`, `remote`,
`description`), returns a populated `Enrichment` value. The provider SHALL instruct
the LLM with the controlled vocabularies from the phase-1 contract so that enum
fields are constrained to their allowed values. Fields not determinable from the
input SHALL be omitted, not guessed.

#### Scenario: Description fields are mapped into the contract

- **WHEN** the provider is given a job whose description states "Senior Go engineer,
  fully remote, €70k–90k/year"
- **THEN** it returns an `Enrichment` with `seniority=senior`, `work_mode=remote`,
  `salary_min=70000`, `salary_max=90000`, `salary_currency=EUR`,
  `salary_period=year`, and `skills` including `go`

#### Scenario: Unstated fields are omitted

- **WHEN** a job description says nothing about visa sponsorship or company size
- **THEN** the returned `Enrichment` leaves `visa_sponsorship`, `company_size`, and
  every other unstated field absent rather than filled with a guess

### Requirement: The LLM endpoint is configured provider-agnostically

The system SHALL configure the enrichment LLM from three provider-neutral settings:
`LLM_BASE_URL` (an OpenAI-compatible API endpoint — e.g. a LiteLLM gateway or a
Chinese model provider), `LLM_API_KEY` (the credential), and `LLM_MODEL` (the model
id). No provider name, vendor-specific key, or default model SHALL be hard-coded.
The enrichment command SHALL fail with a clear error when any of the three is unset.

#### Scenario: Endpoint and model come from config

- **WHEN** `LLM_BASE_URL`, `LLM_API_KEY`, and `LLM_MODEL` are set
- **THEN** the provider calls that endpoint with that model, with no provider name
  baked into the code

#### Scenario: Switching provider needs no code change

- **WHEN** `LLM_BASE_URL` and `LLM_MODEL` are changed to a different OpenAI-compatible
  provider
- **THEN** the enrichment run targets the new provider without a code change or
  rebuild

#### Scenario: Missing configuration fails fast

- **WHEN** any of `LLM_BASE_URL`, `LLM_API_KEY`, or `LLM_MODEL` is unset
- **THEN** the enrichment command exits with an error naming the missing setting and
  enriches no jobs

### Requirement: Queue entries are claimed safely under concurrency

The system SHALL claim a bounded batch of outbox entries that are not dead-lettered
and not currently leased, using `FOR UPDATE SKIP LOCKED`, stamping `claimed_at` on
each claimed entry. Concurrent claimers SHALL receive disjoint entries. An entry whose
`claimed_at` is older than the lease duration SHALL become claimable again, so a
crashed or stalled worker's entries are reclaimed without a separate process.

#### Scenario: Concurrent workers get disjoint entries

- **WHEN** two enrichment runs claim a batch at the same time
- **THEN** no outbox entry is handed to both runs

#### Scenario: A stalled claim is reclaimed after the lease

- **WHEN** an entry was claimed but its `claimed_at` is older than the lease duration
  and it was never completed
- **THEN** a subsequent claim is allowed to pick it up again

#### Scenario: Dead-lettered entries are not claimed

- **WHEN** an entry has a non-null `failed_at`
- **THEN** it is not returned by a claim

### Requirement: Validated write-back stamps provenance and removes the queue entry

When extraction passes `Enrichment.Validate`, the system SHALL, in one transaction,
write the payload to the job's `enrichment` column, set `enriched_at` to the write
time, set `enrichment_version` to the entry's `target_version`, and delete the outbox
entry. The write SHALL NOT modify any raw source field (`title`, `company`,
`location`, `remote`, `description`, `posted_at`, `company_slug`).

#### Scenario: Successful enrichment is written and dequeued

- **WHEN** a claimed job is enriched and the payload validates
- **THEN** the job's `enrichment` is set, `enriched_at` is non-null,
  `enrichment_version` equals the entry's `target_version`, the outbox entry is gone,
  and the job's raw source fields are unchanged

### Requirement: Repeated failures are retried then dead-lettered

An extraction that fails validation SHALL be retried at most once within the same
attempt before the attempt is counted as failed. On a failed attempt the system SHALL
increment the entry's `attempts` and record the error, leaving its lease in place so
the entry is retried on a later run after the lease expires (never reprocessed within
the same run); once `attempts` reaches the configured maximum the entry SHALL be
dead-lettered (`failed_at` set) and no invalid payload SHALL ever be written to `jobs`.

#### Scenario: A transient failure is retried on a later run

- **WHEN** enriching an entry fails once (validation or LLM error) and its attempts are
  below the maximum
- **THEN** the job is left unenriched, the entry's `attempts` is incremented, and the
  entry becomes eligible to be claimed again only after its lease expires

#### Scenario: A persistently failing entry is dead-lettered

- **WHEN** an entry's attempts reach the configured maximum
- **THEN** its `failed_at` is set, it is no longer claimed, and the job's `enrichment`
  was never written with an invalid value

### Requirement: A batch command runs the enrichment process

The system SHALL provide a standalone command (`cmd/enrich`) that connects to the
database, enqueues pending jobs, claims and drains a batch (enriching and writing back
each), and reports how many entries were enriched, failed, and dead-lettered. A
failure on one entry SHALL NOT abort the run.

#### Scenario: A run reports its outcome

- **WHEN** `cmd/enrich` processes a batch with some enrichable and some failing
  entries
- **THEN** it writes the enrichable ones, advances the failing ones' attempts, and
  exits reporting the enriched / failed / dead-lettered counts

#### Scenario: One failing entry does not abort the run

- **WHEN** enriching a single entry returns an error (e.g. an LLM call fails)
- **THEN** that entry is recorded as a failed attempt and the run proceeds to the
  remaining entries
