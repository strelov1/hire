## MODIFIED Requirements

### Requirement: Marking a job applied

The system SHALL let an authenticated user mark a job as applied, idempotently,
and SHALL seed `stage = 'applied'` when the stage is currently unset (an
already-set stage is left untouched). Authentication MAY be by session cookie or
by API key; either identifies the acting user identically. Marking applied sets
`applied_at`; it works whether or not a view was recorded first, and repeating it
does not create a duplicate or error. The endpoint SHALL return the updated
interaction record.

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

#### Scenario: Applying seeds the initial stage

- **WHEN** an authenticated user applies to a job whose `stage` is unset
- **THEN** the interaction's `stage` becomes `applied`
- **AND** applying again, or after the stage has been advanced, leaves the
  existing stage unchanged

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

## ADDED Requirements

### Requirement: Tracking application stage and notes

The system SHALL let an authenticated user set an application's `stage` and/or
free-text `notes` via `PATCH /api/v1/jobs/:slug/track`, authenticated by session
cookie or API key. The body carries optional `stage` and `notes`, of which at
least one MUST be present (else `400`). The endpoint SHALL upsert the
`(user, job)` interaction (creating it if absent) and apply a partial update — a
field omitted from the body leaves its stored column unchanged. A provided
`stage` MUST be one of the controlled vocabulary values, and an unknown value
SHALL be rejected with `400`. The endpoint SHALL return the updated interaction
record.

The stage vocabulary SHALL be the active stages `applied`, `screening`,
`responded`, `interview`, `offer` and the terminal stages `accepted`,
`rejected`, `withdrawn`. Transitions are unrestricted: any valid stage may be set
from any other.

#### Scenario: Set a stage

- **WHEN** an authenticated user sends `PATCH /api/v1/jobs/:slug/track` with
  `{"stage":"interview"}` for a job they have not interacted with
- **THEN** the system creates the interaction with `stage = interview` and
  responds `200` with the record

#### Scenario: Set notes without changing the stage

- **WHEN** the user sends `{"notes":"recruiter called Friday"}` with no `stage`
- **THEN** `notes` is updated and the existing `stage` is left unchanged

#### Scenario: Unknown stage is rejected

- **WHEN** the user sends `{"stage":"banana"}`
- **THEN** the system responds `400` and changes nothing

#### Scenario: Empty track is rejected

- **WHEN** the user sends `track` with neither `stage` nor `notes`
- **THEN** the system responds `400`

#### Scenario: Track authenticated by an API key

- **WHEN** a `track` request carries a valid `Authorization: Bearer <key>` and no
  cookie
- **THEN** the stage/notes are set for the key's owning user exactly as a cookie
  session would

#### Scenario: Track requires authentication

- **WHEN** a `track` request carries neither a valid cookie nor a valid API key
- **THEN** the system responds `401` and changes nothing

### Requirement: Interaction records carry stage and notes

Interaction records SHALL carry the application's `stage` and `notes` (null when
unset) — on the view, apply, save, unsave, and track responses and on every
my-jobs listing row. No other field of the existing interaction or my-jobs shapes
changes.

#### Scenario: Stage and notes on the interaction response

- **WHEN** any per-user interaction endpoint returns the interaction record
- **THEN** the JSON includes `stage` and `notes` (null when unset) alongside
  `job_id`, `viewed_at`, `saved_at`, `applied_at`

#### Scenario: Stage and notes on the my-jobs listing

- **WHEN** `GET /api/v1/me/jobs` returns the user's tracked jobs
- **THEN** each row includes the job's `stage` and `notes`

### Requirement: SPA shows and edits application stage and notes

The web SPA's My Jobs page SHALL, for a signed-in user, show each tracked job's
`stage` as a humanized badge when set, let the user change the stage from a
control offering the stage vocabulary (persisting via the track endpoint), and
let the user edit `notes` inline (persisting via the track endpoint). A signed-out
user SHALL see no such controls.

#### Scenario: Change a stage

- **WHEN** a signed-in user selects a new stage for a job on My Jobs
- **THEN** the SPA persists it via the track endpoint and reflects the new stage

#### Scenario: Edit notes

- **WHEN** a signed-in user edits a job's notes and the field loses focus
- **THEN** the SPA persists the notes via the track endpoint
