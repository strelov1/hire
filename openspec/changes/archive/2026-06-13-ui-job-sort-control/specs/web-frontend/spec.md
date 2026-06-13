# web-frontend Specification (delta)

## ADDED Requirements

### Requirement: Jobs browse sort control

The jobs browse UI SHALL provide a sort control offering two options: **Date
posted** (the source's `posted_at`) and **Recently added** (`created_at`), each
ordered newest first. Selecting an option SHALL refetch the list ordered by that
field. The selection SHALL be mirrored into the URL query string (`?sort=`,
alongside the existing filter params) so it survives reload, sharing, and
back/forward navigation. The default selection SHALL be **Date posted**, and the
URL SHALL omit `?sort=` while the default is active (kept clean, like an empty
search query).

#### Scenario: Default sort is by posting date

- **WHEN** a user opens the jobs page with no `sort` in the URL
- **THEN** the control shows "Date posted" and the list is ordered by
  `posted_at` descending

#### Scenario: User switches to recently added

- **WHEN** a user selects "Recently added"
- **THEN** the list is refetched ordered by `created_at` descending and the URL
  query string is updated to include `sort=created_at`

#### Scenario: Sort restored from the URL

- **WHEN** a user opens the jobs page with `?sort=created_at` directly or via
  back/forward
- **THEN** the control is preset to "Recently added" and the list is ordered by
  `created_at` descending
