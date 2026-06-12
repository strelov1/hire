## ADDED Requirements

### Requirement: Remote type filter facet

The frontend job-search filter UI SHALL offer a curated "Remote type" facet,
rendered as pills under the "Work format" facet, that filters on the search API's
`remote_type` parameter. Its options SHALL be a curated, extensible set of the
most relevant remote destinations (Global, Russia, Europe, USA), each mapping to
a lowercased `remote_type` code (`global`, `ru`, `eu`, `us`). The facet SHALL
support exclusion like the other facets. The facet's option values SHALL mirror
the codes produced by the backend's derived `remote_type` field.

#### Scenario: Filtering by a remote type

- **WHEN** a user selects the "Europe" pill in the Remote type facet
- **THEN** the search request carries `remote_type=eu` and the results are jobs
  whose remote reach includes Europe

#### Scenario: Excluding a remote type

- **WHEN** a user excludes the "USA" pill
- **THEN** the search request excludes `remote_type=us` and such jobs are omitted

## MODIFIED Requirements

### Requirement: Jobs list with pagination

The frontend SHALL present a list of jobs from `GET /api/v1/jobs`, showing for
each job its title, company, location, work arrangement (and, for remote roles,
its reach), source, and posted date, and SHALL paginate using the API's
`limit`/`offset` driven by `meta.total`. The work arrangement SHALL be derived
from `enrichment.work_mode`; for a remote role the reach SHALL be shown as
`Global` (when `remote_scope=global`), the region(s) (when `regional`), or the
country code(s) (when `national`). The frontend SHALL NOT rely on a raw `remote`
field (it is no longer in the API).

#### Scenario: Jobs are listed

- **WHEN** a user opens the jobs route `/`
- **THEN** a page of jobs is fetched and rendered as rows, each linking to its
  job detail

#### Scenario: User loads more jobs

- **WHEN** more jobs exist than the current page (`offset + limit < meta.total`)
- **THEN** a control lets the user fetch and view the next page

#### Scenario: A global-remote job shows its reach explicitly

- **WHEN** a listed job has `work_mode=remote` and `remote_scope=global`
- **THEN** its row shows a "Global" reach indicator rather than a bare "Remote"
  with no reach

### Requirement: Job detail

The frontend SHALL show a single job from `GET /api/v1/jobs/:id` with its title,
company link, work-arrangement/source badges, posted date, description, and a
link to the external posting URL. For a remote role the work-arrangement badge
SHALL convey reach (global, region(s), or country(ies)) from
`enrichment.work_mode`/`remote_scope`/`regions`/`countries` rather than a raw
`remote` flag.

#### Scenario: Job detail is shown

- **WHEN** a user navigates to `/jobs/:id`
- **THEN** the job's fields are fetched and displayed, with an "Apply" link
  pointing to `job.url`

#### Scenario: Missing job

- **WHEN** the API returns 404 for the requested id
- **THEN** the view shows an error state instead of broken content

#### Scenario: Remote reach is shown on detail

- **WHEN** a job has `work_mode=remote` and `remote_scope=regional`,
  `regions=[EU]`
- **THEN** the detail view conveys a Europe reach rather than only "Remote"
