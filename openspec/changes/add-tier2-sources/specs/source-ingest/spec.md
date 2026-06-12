## ADDED Requirements

### Requirement: Personio, Pinpoint, Rippling, and BambooHR are registered providers

The system SHALL register `personio`, `pinpoint`, `rippling`, and `bamboohr` adapters so
boards on these platforms can be listed in `sources.yml`. Each adapter SHALL yield the
normalized job shape (at least title, url, location, remote flag, description, and the
platform's native posting id) with the `description` as sanitized HTML assembled from the
platform's authoritative HTML field(s), consistent with the existing adapters. An adapter
whose list endpoint omits the description SHALL fetch each posting's detail with bounded
concurrency rather than yield an empty body, and a single failed detail SHALL drop only
that posting rather than abort the board.

#### Scenario: Personio XML feed is crawled in one request

- **WHEN** `sources.yml` lists a board with provider `personio`
- **THEN** the adapter fetches the board's `…jobs.personio.com/xml` feed in one request and
  yields each `<position>` with a sanitized HTML description assembled from its inline
  `jobDescriptions`, and a job URL built from the board and position id

#### Scenario: Pinpoint board carries the body inline

- **WHEN** a `pinpoint` board is crawled
- **THEN** the adapter fetches the board's `…/postings.json` in one request and yields each
  posting with a sanitized HTML description assembled from its inline body sections

#### Scenario: Rippling posting gains its description from detail

- **WHEN** a `rippling` board is crawled
- **THEN** the adapter fetches the board's job list and, per posting, fetches its detail with
  bounded concurrency to obtain the role description (excluding the company boilerplate),
  still yielding the normalized job shape

#### Scenario: BambooHR posting gains its description from detail

- **WHEN** a `bamboohr` board is crawled
- **THEN** the adapter fetches `…/careers/list` and, per posting, fetches `…/careers/{id}/detail`
  with bounded concurrency to obtain the description, still yielding the normalized job shape

#### Scenario: A failed detail request drops only that posting

- **WHEN** a detail-fetching provider's board lists several postings and one posting's detail
  request fails
- **THEN** the failed posting is skipped and every other posting is still yielded, without
  aborting the board

#### Scenario: A board with no open postings yields no jobs without error

- **WHEN** any of these providers' feeds returns an empty posting list for a configured board
- **THEN** the adapter yields zero jobs and returns no error, so the board is simply skipped
