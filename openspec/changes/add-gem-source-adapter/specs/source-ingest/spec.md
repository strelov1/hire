## ADDED Requirements

### Requirement: Gem is a registered provider

The system SHALL register a `gem` adapter so boards on the Gem platform (jobs.gem.com)
can be listed in `sources.yml`. The adapter SHALL treat the configured `board` value as
the Gem **vanity path** and pass it verbatim as the GraphQL `boardId`. It SHALL fetch
postings from the public GraphQL endpoint `POST https://jobs.gem.com/api/public/graphql`
using the `JobBoardList(boardId)` operation, and — because that list carries no
description — SHALL fetch each posting's body via the `ExternalJobPostingQuery(boardId,
extId)` operation with bounded concurrency. A single failed detail request SHALL drop
only that posting rather than abort the board. The adapter SHALL yield the normalized job
shape (at least title, url, location, remote flag, description, and the platform's native
posting id), with the `description` as sanitized HTML assembled from the posting's
`descriptionHtml` field, consistent with the existing adapters.

#### Scenario: Gem board is crawled list-then-detail

- **WHEN** `sources.yml` lists a board with provider `gem` and a vanity-path `board`
- **THEN** the adapter calls `JobBoardList` with that vanity path as `boardId`, and per
  returned posting calls `ExternalJobPostingQuery` with the posting's `extId` to obtain a
  sanitized HTML description, yielding each as the normalized job shape with
  `external_id` set to the posting's `extId`

#### Scenario: Remote is taken from Gem's structured flags

- **WHEN** a Gem posting reports a location with `isRemote: true` or a `job.locationType`
  of `REMOTE`
- **THEN** the adapter yields the job with its remote flag set, without relying on a
  free-text location match

#### Scenario: Posting is dated from its first-published timestamp

- **WHEN** a Gem posting carries a `firstPublishedTsSec` Unix-seconds timestamp
- **THEN** the adapter yields the job with `posted_at` derived from that timestamp, and
  yields a nil `posted_at` when the timestamp is absent or zero

#### Scenario: A failed detail request drops only that posting

- **WHEN** a Gem board lists several postings and one posting's
  `ExternalJobPostingQuery` request fails
- **THEN** the failed posting is skipped and every other posting is still yielded, without
  aborting the board

#### Scenario: A board with no open postings yields no jobs without error

- **WHEN** a Gem board returns an empty `jobPostings` list
- **THEN** the adapter yields zero jobs and returns no error, so the board is simply
  skipped
