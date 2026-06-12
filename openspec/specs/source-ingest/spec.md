# source-ingest Specification

## Purpose
TBD - created by archiving change add-source-ingest-pipeline. Update Purpose after archive.
## Requirements
### Requirement: Jobs enter the catalogue through modular source adapters

The system SHALL ingest jobs through `Source` adapters, each implementing exactly
one job-source platform. An adapter SHALL expose its provider key and SHALL fetch
all current postings for one configured board. Adapters SHALL be assembled into a
provider-keyed registry by a single explicit constructor, so that adding a platform
is a new adapter file plus one registration line and requires no change to the
pipeline.

An adapter SHALL prefer the platform's list endpoint, but MAY perform per-posting
detail requests and paginate the list when the list endpoint does not carry the
full posting (for example, when it omits the description). Such detail requests
SHALL be bounded so a single board cannot issue unbounded concurrent requests.

#### Scenario: Adapter is dispatched by provider key

- **WHEN** a configured board names provider `greenhouse`
- **THEN** the pipeline dispatches that board to the registered `greenhouse` adapter
  and uses the postings it returns

#### Scenario: Adapter maps a posting to the normalized job shape

- **WHEN** an adapter fetches a board and the platform returns a posting
- **THEN** the adapter yields a job carrying at least title, url, location, remote
  flag, description, and the platform's native posting id

#### Scenario: Adapter fetches detail when the list lacks the description

- **WHEN** a platform's list endpoint returns postings without a description (e.g. SmartRecruiters)
- **THEN** the adapter paginates the list and fetches each posting's detail to obtain
  the description, still yielding the normalized job shape

### Requirement: Boards to crawl are configured in a file

The system SHALL read the set of boards to crawl from a `sources.yml` file at ingest
startup, each entry naming a `company`, a `provider`, and a `board`. A configured
entry whose `provider` has no registered adapter SHALL cause the ingest command to
fail fast at startup rather than silently skip the board.

#### Scenario: Configured boards are loaded

- **WHEN** `sources.yml` lists a board with `company`, `provider`, and `board`
- **THEN** the ingest run includes that board, dispatched to the named provider

#### Scenario: Unknown provider fails fast

- **WHEN** `sources.yml` names a `provider` with no registered adapter
- **THEN** the ingest command exits with an error naming the unknown provider and
  ingests nothing

### Requirement: Ingest writes jobs through a normalized, namespaced write path

The pipeline SHALL persist each fetched posting via the existing job write path. It
SHALL set `source` to the board's provider, derive `company_slug` from the company
name using the existing slug normalization, and set `external_id` to the namespaced
form `"<board>:<native-posting-id>"`. Namespacing SHALL guarantee that two companies
on the same platform sharing a native posting id do not collide on the dedup key
`UNIQUE (source, external_id)`.

#### Scenario: External id is namespaced by board

- **WHEN** a posting with native id `42` is ingested for board `cohere` on provider
  `greenhouse`
- **THEN** the stored job has `source = "greenhouse"` and
  `external_id = "cohere:42"`

#### Scenario: Same native id on different boards does not collide

- **WHEN** two boards on the same provider each return a posting with native id `42`
- **THEN** both jobs are stored as distinct rows, differing in `external_id`

#### Scenario: Re-ingest of the same posting updates in place

- **WHEN** a posting already in the catalogue is ingested again with an edited title
- **THEN** the existing row is updated rather than duplicated, keyed on
  `(source, external_id)`

### Requirement: Re-ingest preserves existing enrichment

The ingest write path SHALL NOT write or overwrite a job's enrichment payload or
provenance. Re-ingesting a job that has already been enriched SHALL leave its
`enrichment`, `enriched_at`, and `enrichment_version` unchanged, so that source
re-ingestion never wipes enrichment produced by the enrichment worker.

#### Scenario: Enrichment survives a re-ingest

- **WHEN** an already-enriched job is re-ingested with updated source fields
- **THEN** the job's source fields update but its `enrichment`, `enriched_at`, and
  `enrichment_version` are unchanged

### Requirement: A source failure is isolated from the rest of the run

The ingest run SHALL process boards with bounded concurrency and SHALL isolate
failures: a board whose fetch or parse errors SHALL be recorded and skipped without
aborting the run or preventing the remaining boards from being ingested.

#### Scenario: One failing board does not abort the run

- **WHEN** one configured board's fetch errors and the others succeed
- **THEN** the failing board is recorded as failed and every other board is still
  ingested

### Requirement: A standalone command runs ingest on a schedule

The system SHALL provide a standalone `cmd/ingest` binary that loads configuration,
runs every configured board once with bounded concurrency, reports how many jobs
were ingested and how many sources failed, and exits — suitable for scheduled
execution.

#### Scenario: Ingest command runs a bounded batch and exits

- **WHEN** the ingest command is run
- **THEN** it processes every configured board once and exits after reporting the
  ingested and failed counts

### Requirement: Adapter descriptions are sanitized HTML

Each adapter SHALL yield the job `description` as sanitized HTML assembled from the platform's authoritative HTML field(s), so the stored value is safe to render directly in a browser without further escaping. An adapter SHALL NOT yield raw or entity-encoded source markup, and SHALL NOT rely on a platform plain-text field that the platform may leave empty or partial. Sanitization SHALL run server-side before the description is persisted, stripping scripts, event handlers, and other active content while preserving structural formatting (headings, paragraphs, lists, emphasis, links).

#### Scenario: Greenhouse entity-encoded HTML is decoded and sanitized

- **WHEN** a greenhouse posting returns `content` as entity-encoded HTML (e.g. `&lt;h2&gt;Role&lt;/h2&gt;`)
- **THEN** the adapter yields a description whose entities are decoded to real markup and then sanitized, so the stored value contains `<h2>Role</h2>` rather than the encoded entities

#### Scenario: Lever multi-field body is assembled, not truncated

- **WHEN** a lever posting splits its body across `description`, one or more `lists` (each with a heading `text` and HTML `content`), and `additional`
- **THEN** the adapter yields a description that concatenates the opening `description`, each list as a heading followed by its content, and the closing `additional` — even when `descriptionPlain` is empty

#### Scenario: Ashby uses the HTML field

- **WHEN** an ashby posting exposes both `descriptionHtml` and `descriptionPlain`
- **THEN** the adapter yields the sanitized `descriptionHtml`, preserving its formatting

#### Scenario: Active content is stripped

- **WHEN** a source posting's HTML contains a `<script>` tag or an inline event handler (e.g. `onclick`)
- **THEN** the persisted description contains neither the script nor the event handler, while its safe structural markup is retained

### Requirement: Workable, Recruitee, and SmartRecruiters are registered providers

The system SHALL register `workable`, `recruitee`, and `smartrecruiters` adapters so
boards on these platforms can be listed in `sources.yml`. Each adapter SHALL yield the
job description as sanitized HTML assembled from the platform's authoritative HTML
field(s), consistent with the existing adapters.

#### Scenario: Workable board is crawled

- **WHEN** `sources.yml` lists a board with provider `workable`
- **THEN** the adapter fetches that account's jobs in one request and yields each with a
  sanitized HTML description from the inline `description` field

#### Scenario: Recruitee description and requirements are combined

- **WHEN** a recruitee offer carries separate `description` and `requirements` HTML
- **THEN** the adapter yields one sanitized HTML description combining both

#### Scenario: SmartRecruiters posting gains its description from detail

- **WHEN** a smartrecruiters board is crawled
- **THEN** the adapter paginates the postings list and, per posting, fetches its detail
  and yields a sanitized HTML description assembled from `jobAd.sections`
