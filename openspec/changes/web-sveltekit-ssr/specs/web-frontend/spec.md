## MODIFIED Requirements

### Requirement: Jobs list with pagination

The frontend SHALL present a list of jobs from `GET /api/v1/jobs`, showing for
each job its title, company, location, work arrangement (and, for remote roles,
its reach), source, and posted date, and SHALL paginate using the API's
`limit`/`offset` driven by `meta.total`. The work arrangement SHALL be derived
from `enrichment.work_mode`; for a remote role the reach SHALL be shown from
`enrichment.regions` (e.g. `Global`, `Europe`). The frontend SHALL NOT rely on a
raw `remote` field (it is no longer in the API). The first page of the list
SHALL be **server-rendered** — its rows present in the initial HTML — and then
hydrate on the client for subsequent interaction.

#### Scenario: Jobs are listed

- **WHEN** a user opens the jobs route `/jobs`
- **THEN** the server returns HTML already containing the first page of job rows,
  each linking to its job detail

#### Scenario: User loads more jobs

- **WHEN** more jobs exist than the current page (`offset + limit < meta.total`)
- **THEN** a control lets the user fetch and view the next page

#### Scenario: A global-remote job shows its reach explicitly

- **WHEN** a listed job has `work_mode=remote` and `regions=[global]`
- **THEN** its row shows a "Global" reach indicator rather than a bare "Remote"
  with no reach

### Requirement: Job detail

The frontend SHALL show a single job from the public job API at the route
`/jobs/:slug` with its title, company link, work-arrangement/source badges,
posted date, description, and a link to the external posting URL. For a remote
role the displayed facets SHALL convey reach from `enrichment.regions` rather
than a raw `remote` flag. The page SHALL be **server-rendered** — the job's
fields present in the initial HTML — and then hydrate on the client.

#### Scenario: Job detail is shown

- **WHEN** a user navigates to `/jobs/:slug`
- **THEN** the server returns HTML already containing the job's fields, with an
  "Apply" link pointing to `job.url`

#### Scenario: Missing job

- **WHEN** the API returns 404 for the requested slug
- **THEN** the view shows an error state instead of broken content

#### Scenario: Remote reach is shown on detail

- **WHEN** a job has `work_mode=remote` and `regions=[eu]`
- **THEN** the detail view conveys a Europe reach rather than only "Remote"

### Requirement: Company detail

The frontend SHALL show a single company from `GET /api/v1/companies/:slug`
together with its jobs, reusing the same job row presentation as the jobs list.
The page SHALL be **server-rendered** — the company info and its jobs present in
the initial HTML — and then hydrate on the client.

#### Scenario: Company detail is shown

- **WHEN** a user navigates to `/companies/:slug`
- **THEN** the server returns HTML already containing the company info and its
  jobs
