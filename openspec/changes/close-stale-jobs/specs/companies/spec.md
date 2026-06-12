## MODIFIED Requirements

### Requirement: Company surfaces count and list only open jobs

The company list's `job_count` and the company detail's jobs SHALL include only
open jobs (`closed_at IS NULL`).

#### Scenario: Closed job leaves the company page

- **WHEN** a company's job is closed
- **THEN** the company detail no longer lists it and the company's `job_count`
  no longer counts it
