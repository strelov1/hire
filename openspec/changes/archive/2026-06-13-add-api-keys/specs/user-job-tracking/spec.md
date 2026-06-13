## MODIFIED Requirements

### Requirement: Recording a job view

The system SHALL let an authenticated user record that they viewed a job, keyed
by `(user, job)`, idempotently. Authentication MAY be by session cookie or by API
key; either identifies the acting user identically. The first view creates the
interaction; a repeat view refreshes its timestamp without creating a duplicate.
The endpoint SHALL return the interaction record, including whether the job has
been applied to.

#### Scenario: First view by a signed-in user

- **WHEN** an authenticated user sends `POST /api/v1/jobs/:id/view` for a job
  they have not interacted with before
- **THEN** the system creates a `user_jobs` row with `viewed_at` set and
  `applied_at` null
- **AND** responds `200` with `{"data": {job_id, viewed_at, applied_at: null}}`

#### Scenario: Repeat view does not duplicate

- **WHEN** an authenticated user views the same job a second time
- **THEN** the existing row's `viewed_at` is refreshed
- **AND** no second row is created
- **AND** the response carries the existing `applied_at` value unchanged

#### Scenario: View requires authentication

- **WHEN** a request to `POST /api/v1/jobs/:id/view` carries neither a valid auth
  cookie nor a valid API key
- **THEN** the system responds `401` and records nothing

#### Scenario: View authenticated by an API key

- **WHEN** a request to `POST /api/v1/jobs/:id/view` carries a valid
  `Authorization: Bearer <key>` and no cookie
- **THEN** the system records the view for the key's owning user exactly as a
  cookie session would and responds `200` with the interaction record

#### Scenario: View with a non-numeric id

- **WHEN** an authenticated user sends `POST /api/v1/jobs/:id/view` with an `:id`
  that is not a valid job id
- **THEN** the system responds with a client error (`400`) and records nothing

### Requirement: Marking a job applied

The system SHALL let an authenticated user mark a job as applied, idempotently.
Authentication MAY be by session cookie or by API key; either identifies the
acting user identically. Marking applied sets `applied_at`; it works whether or
not a view was recorded first, and repeating it does not create a duplicate or
error. The endpoint SHALL return the updated interaction record.

#### Scenario: Mark applied after viewing

- **WHEN** an authenticated user who has viewed a job sends
  `POST /api/v1/jobs/:id/apply`
- **THEN** the job's `applied_at` is set
- **AND** the response is `200` with `{"data": {job_id, viewed_at, applied_at}}`
  where `applied_at` is non-null

#### Scenario: Mark applied is idempotent

- **WHEN** an authenticated user marks the same job applied twice
- **THEN** the row is updated in place each time
- **AND** no duplicate row is created and no error is returned

#### Scenario: Apply requires authentication

- **WHEN** a request to `POST /api/v1/jobs/:id/apply` carries neither a valid auth
  cookie nor a valid API key
- **THEN** the system responds `401` and records nothing

#### Scenario: Apply authenticated by an API key

- **WHEN** a request to `POST /api/v1/jobs/:id/apply` carries a valid
  `Authorization: Bearer <key>` and no cookie
- **THEN** the system marks the job applied for the key's owning user exactly as a
  cookie session would and responds `200` with the updated interaction record

#### Scenario: Apply to a non-existent job

- **WHEN** an authenticated user sends `POST /api/v1/jobs/:id/apply` with a
  numeric `:id` that has no corresponding job row
- **THEN** the foreign-key violation surfaces as `404`, not `500`
