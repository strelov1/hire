## ADDED Requirements

### Requirement: API permits cross-origin browser access

The HTTP API SHALL respond with CORS headers that allow a browser running on a
different origin to call the read endpoints, so the frontend can fetch data
directly without a proxy.

#### Scenario: Browser preflight is allowed

- **WHEN** a browser sends an `OPTIONS` preflight to `/api/v1/jobs` with an
  `Origin` header
- **THEN** the response includes `Access-Control-Allow-Origin` matching the
  configured frontend origin and the request succeeds

#### Scenario: Cross-origin GET returns data

- **WHEN** the frontend issues a cross-origin `GET /api/v1/jobs`
- **THEN** the response carries `Access-Control-Allow-Origin` and the JSON body
  is readable by the browser

### Requirement: Jobs list with pagination

The frontend SHALL present a list of jobs from `GET /api/v1/jobs`, showing for
each job its title, company, location, remote status, source, and posted date,
and SHALL paginate using the API's `limit`/`offset` driven by `meta.total`.

#### Scenario: Jobs are listed

- **WHEN** a user opens the jobs route `/`
- **THEN** a page of jobs is fetched and rendered as rows, each linking to its
  job detail

#### Scenario: User loads more jobs

- **WHEN** more jobs exist than the current page (`offset + limit < meta.total`)
- **THEN** a control lets the user fetch and view the next page

### Requirement: Job detail

The frontend SHALL show a single job from `GET /api/v1/jobs/:id` with its title,
company link, remote/source badges, posted date, description, and a link to the
external posting URL.

#### Scenario: Job detail is shown

- **WHEN** a user navigates to `/jobs/:id`
- **THEN** the job's fields are fetched and displayed, with an "Apply" link
  pointing to `job.url`

#### Scenario: Missing job

- **WHEN** the API returns 404 for the requested id
- **THEN** the view shows an error state instead of broken content

### Requirement: Companies list

The frontend SHALL present companies from `GET /api/v1/companies`, showing each
company's name and its job count, with each row linking to the company detail.

#### Scenario: Companies are listed

- **WHEN** a user opens `/companies`
- **THEN** a page of companies is fetched and rendered with job counts

### Requirement: Company detail

The frontend SHALL show a single company from `GET /api/v1/companies/:slug`
together with its jobs, reusing the same job row presentation as the jobs list.

#### Scenario: Company detail is shown

- **WHEN** a user navigates to `/companies/:slug`
- **THEN** the company info and its jobs are fetched and displayed

### Requirement: Light and dark theme

The frontend SHALL support light, dark, and system themes, applying dark mode via
a `.dark` class on the document root, persisting the choice in localStorage, and
tracking `prefers-color-scheme` when in system mode.

#### Scenario: User toggles theme

- **WHEN** a user activates the theme toggle
- **THEN** the interface switches between light and dark and the choice persists
  across reloads

#### Scenario: System mode follows OS preference

- **WHEN** the theme is set to system
- **THEN** the effective theme matches the OS `prefers-color-scheme` and updates
  if the OS preference changes

### Requirement: Async load states

Every data-driven view SHALL render distinct loading, empty, and error states so
the user is never shown broken or blank content during or after a fetch.

#### Scenario: Loading state

- **WHEN** a view's request is in flight
- **THEN** a loading indicator is shown until data or an error arrives

#### Scenario: Empty state

- **WHEN** a successful response contains no items
- **THEN** an empty-state message is shown instead of an empty list

#### Scenario: Error state

- **WHEN** a request fails (network or non-2xx)
- **THEN** an error message is shown
