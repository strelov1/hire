## ADDED Requirements

### Requirement: Region (remote reach) filter facet

The frontend job-search filter UI SHALL offer a curated "Region" facet, rendered
as pills under the "Work format" facet, that filters on the search API's
`regions` parameter. Its options SHALL be a curated, extensible subset of the
reach vocabulary (Global, Russia, Europe, USA), each mapping to a `regions` code
(`global`, `ru`, `eu`, `us`). The facet SHALL support exclusion like the other
facets. The facet's option values SHALL be codes from the backend's `regions`
vocabulary.

#### Scenario: Filtering by a region

- **WHEN** a user selects the "Europe" pill in the Region facet
- **THEN** the search request carries `regions=eu` and the results are jobs whose
  reach includes Europe

#### Scenario: Excluding a region

- **WHEN** a user excludes the "USA" pill
- **THEN** the search request excludes `regions=us` and such jobs are omitted

## MODIFIED Requirements

### Requirement: Jobs list with pagination

The frontend SHALL present a list of jobs from `GET /api/v1/jobs`, showing for
each job its title, company, location, work arrangement (and, for remote roles,
its reach), source, and posted date, and SHALL paginate using the API's
`limit`/`offset` driven by `meta.total`. The work arrangement SHALL be derived
from `enrichment.work_mode`; for a remote role the reach SHALL be shown from
`enrichment.regions` (e.g. `Global`, `Europe`). The frontend SHALL NOT rely on a
raw `remote` field (it is no longer in the API).

#### Scenario: Jobs are listed

- **WHEN** a user opens the jobs route `/`
- **THEN** a page of jobs is fetched and rendered as rows, each linking to its
  job detail

#### Scenario: User loads more jobs

- **WHEN** more jobs exist than the current page (`offset + limit < meta.total`)
- **THEN** a control lets the user fetch and view the next page

#### Scenario: A global-remote job shows its reach explicitly

- **WHEN** a listed job has `work_mode=remote` and `regions=[global]`
- **THEN** its row shows a "Global" reach indicator rather than a bare "Remote"
  with no reach

### Requirement: Job detail

The frontend SHALL show a single job from `GET /api/v1/jobs/:id` with its title,
company link, work-arrangement/source badges, posted date, description, and a
link to the external posting URL. For a remote role the displayed facets SHALL
convey reach from `enrichment.regions` rather than a raw `remote` flag.

#### Scenario: Job detail is shown

- **WHEN** a user navigates to `/jobs/:id`
- **THEN** the job's fields are fetched and displayed, with an "Apply" link
  pointing to `job.url`

#### Scenario: Missing job

- **WHEN** the API returns 404 for the requested id
- **THEN** the view shows an error state instead of broken content

#### Scenario: Remote reach is shown on detail

- **WHEN** a job has `work_mode=remote` and `regions=[eu]`
- **THEN** the detail view conveys a Europe reach rather than only "Remote"
