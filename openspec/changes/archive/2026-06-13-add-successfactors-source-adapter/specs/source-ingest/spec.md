## ADDED Requirements

### Requirement: successfactors is a registered provider

The system SHALL register a `successfactors` adapter so SAP SuccessFactors career sites
can be listed in `sources.yml`. The adapter SHALL treat the configured `board` value as the
career-site host and enumerate jobs from that site's `GET https://<board>/job_sitemap.xml`,
taking each `<url>`'s `<loc>` as the job page URL (with the job's native id as the numeric
segment of that path) and its `<lastmod>` as the posting date. Because the sitemap carries
no description, the adapter SHALL fetch each job page and extract the title and description
from the page's schema.org JobPosting microdata (`itemprop="title"` and
`itemprop="description"`), with bounded concurrency; a single failed page fetch SHALL drop
only that posting rather than abort the board. The adapter SHALL yield the normalized job
shape (at least title, url, remote flag, description, and the platform's native posting id),
with the `description` as sanitized HTML, consistent with the existing adapters. The job
`location` MAY be empty, since SuccessFactors does not expose it in the microdata and
enrichment derives it from the description.

#### Scenario: SuccessFactors board is enumerated from its sitemap

- **WHEN** `sources.yml` lists a board with provider `successfactors` and a career-site host
- **THEN** the adapter fetches `https://<host>/job_sitemap.xml`, and per `<loc>` fetches the
  job page, yielding each as the normalized job shape with `external_id` set to the numeric
  id from the job URL and `posted_at` derived from the entry's `<lastmod>`

#### Scenario: Title and description come from JobPosting microdata

- **WHEN** a SuccessFactors job page is fetched
- **THEN** the adapter yields the job's title from `itemprop="title"` and a sanitized HTML
  description from the inner markup of `itemprop="description"`

#### Scenario: A failed job-page fetch drops only that posting

- **WHEN** a board's sitemap lists several jobs and one job page's fetch fails
- **THEN** the failed posting is skipped and every other posting is still yielded, without
  aborting the board

#### Scenario: An empty sitemap yields no jobs without error

- **WHEN** a board's `job_sitemap.xml` lists no job URLs
- **THEN** the adapter yields zero jobs and returns no error, so the board is simply skipped
