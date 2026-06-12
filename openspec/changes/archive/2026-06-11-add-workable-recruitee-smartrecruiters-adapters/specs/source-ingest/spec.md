## MODIFIED Requirements

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

## ADDED Requirements

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
