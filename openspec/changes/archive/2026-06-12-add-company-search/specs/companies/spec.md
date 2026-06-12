## MODIFIED Requirements

### Requirement: Company list is served without joining jobs

The system SHALL expose `GET /api/v1/companies` returning companies read from the
`companies` table. Job counts, when included, SHALL be computed at query time;
no denormalized counter is required.

The endpoint SHALL accept an optional `q` query parameter that filters companies
by a case-insensitive substring match on the company `name`. An absent or empty
`q` SHALL return the unfiltered list (today's behavior). When `q` is non-empty,
the list `meta.total` SHALL report the count of companies matching `q`, so
pagination over the filtered results is correct.

#### Scenario: Listing companies

- **WHEN** a client requests `GET /api/v1/companies`
- **THEN** the response contains companies under `data` with list `meta`,
  following the existing list response shape

#### Scenario: Searching companies by name

- **WHEN** a client requests `GET /api/v1/companies?q=acme`
- **THEN** the response contains only companies whose name matches `acme`
  case-insensitively, and `meta.total` is the count of matching companies

#### Scenario: Empty query returns the full list

- **WHEN** a client requests `GET /api/v1/companies?q=` (empty or absent)
- **THEN** the response is the unfiltered company list, identical to omitting the
  parameter
