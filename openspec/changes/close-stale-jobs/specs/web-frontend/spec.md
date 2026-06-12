## ADDED Requirements

### Requirement: The job page renders a closed state

When a job view carries `closed_at`, the job page SHALL show that the position is
no longer accepting applications and SHALL NOT render the Apply action. Open jobs
are unaffected.

#### Scenario: Closed job shows the closed state

- **WHEN** a signed-in or anonymous user opens a closed job's page
- **THEN** the page shows a "no longer accepting applications" notice instead of
  the Apply button
