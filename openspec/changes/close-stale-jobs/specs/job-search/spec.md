## MODIFIED Requirements

### Requirement: Searchable jobs index contains only open jobs

The search index SHALL contain documents only for open jobs. The indexer SHALL
skip closed jobs, and a reindex run SHALL remove documents whose jobs have been
closed since the previous run.

#### Scenario: Closed job is dropped on reindex

- **WHEN** a job is closed and a reindex runs
- **THEN** the job's document is removed from the index and no longer matches any search

#### Scenario: Reopened job returns to the index

- **WHEN** a previously closed job is reopened and a reindex runs
- **THEN** the job's document is indexed again
