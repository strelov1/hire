## MODIFIED Requirements

### Requirement: Companies list

The frontend SHALL present companies from `GET /api/v1/companies`, showing each
company's name and its job count, with each row linking to the company detail.

The page SHALL provide a name-search input. Typing SHALL filter the list against
the API's `q` parameter (debounced), and the current query SHALL be mirrored into
the URL query string (`?q=`) so a search survives reload, sharing, and
back/forward navigation. The page SHALL show the count of matching companies and
a distinct empty state when a search matches nothing.

#### Scenario: Companies are listed

- **WHEN** a user opens `/companies`
- **THEN** a page of companies is fetched and rendered with job counts

#### Scenario: User searches companies by name

- **WHEN** a user types a query into the companies search input
- **THEN** the list is refetched filtered by that query and the URL query string
  is updated to `?q=<query>`

#### Scenario: Search restored from the URL

- **WHEN** a user opens `/companies?q=acme` directly or via back/forward
- **THEN** the search input is prefilled with `acme` and the filtered list is
  shown

#### Scenario: Search matches nothing

- **WHEN** a search returns no companies
- **THEN** an empty state ("No matching companies.") is shown instead of an empty
  list
