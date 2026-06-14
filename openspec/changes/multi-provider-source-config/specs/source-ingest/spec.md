## MODIFIED Requirements

### Requirement: Boards to crawl are configured in a file

The system SHALL read the set of boards to crawl from configuration at ingest startup,
each entry naming a `company`, a `provider`, and — for a provider that has a board/tenant
concept — a `board`. An entry's `provider` MAY be set on the entry itself; when it is
absent the provider SHALL default to the configuration file's base name. Because the
provider is resolved per entry, a single file MAY list entries belonging to several
providers (e.g. a shared `custom.yml` of single-source configs). A configured entry whose
resolved `provider` has no registered adapter SHALL cause the ingest command to fail fast
at startup rather than silently skip the board. An entry whose `board` is empty SHALL fail
fast at startup **unless** its provider is a single-company provider that declares it needs
no board (a `boardless` provider), in which case the empty `board` SHALL be accepted.

#### Scenario: Configured boards are loaded

- **WHEN** the configuration lists a board with `company`, `provider`, and `board`
- **THEN** the ingest run includes that board, dispatched to the named provider

#### Scenario: Provider defaults to the file name when the entry omits it

- **WHEN** an entry does not set `provider` and the configuration file's base name is a
  registered provider (e.g. `greenhouse.yml`)
- **THEN** the entry is dispatched to the file-name provider, as before

#### Scenario: A single file lists entries for multiple providers

- **WHEN** one configuration file lists entries that each name their own `provider`
  (e.g. `custom.yml` with `vk` and `ozon` entries)
- **THEN** each entry is dispatched to its own named provider within the same run

#### Scenario: Unknown provider fails fast

- **WHEN** the configuration has an entry whose resolved `provider` has no registered
  adapter (either named explicitly or defaulted from a file name that is not a provider)
- **THEN** the ingest command exits with an error naming the unknown provider and
  ingests nothing

#### Scenario: Empty board fails fast for a board-based provider

- **WHEN** the configuration has an entry with an empty `board` whose provider has a
  board concept (e.g. `greenhouse`)
- **THEN** the ingest command exits with an error naming the company with the empty board

#### Scenario: Empty board is accepted for a boardless provider

- **WHEN** the configuration has an entry with an empty `board` whose provider is a
  single-company `boardless` provider (e.g. `ozon`)
- **THEN** validation accepts the entry and the ingest run includes that board
