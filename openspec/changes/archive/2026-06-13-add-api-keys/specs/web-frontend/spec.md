## ADDED Requirements

### Requirement: API key management page

The SPA SHALL provide an API-keys management page at `/my/api-keys`, reachable
from the authenticated user menu, where a signed-in user can list, create, and
revoke their API keys. The list SHALL show each key's name, display prefix,
created time, last-used time (or "never"), and expiry. Creating a key SHALL
reveal the full plaintext token **once**, with a copy control and a ready-to-run
`curl` example that sends `Authorization: Bearer <key>`, alongside a notice that
the token will not be shown again. Revoking a key SHALL require an explicit
confirmation. The page and its menu entry SHALL be available only to signed-in
users.

#### Scenario: Reaching the page from the user menu

- **WHEN** a signed-in user opens the user menu and selects "API keys"
- **THEN** the SPA navigates to `/my/api-keys` and lists the user's keys with name,
  prefix, created, last-used, and expiry

#### Scenario: Creating a key reveals the secret once

- **WHEN** the user creates a key (name, optional expiry)
- **THEN** the SPA shows the full plaintext token with a copy control, a `curl`
  example using `Authorization: Bearer <key>`, and a "won't be shown again" notice
- **AND** the new key appears in the list

#### Scenario: The secret is not shown again

- **WHEN** the user dismisses the reveal or navigates away and returns
- **THEN** the page shows only the key's metadata (including its prefix), never the
  full token again

#### Scenario: Revoking a key

- **WHEN** the user revokes a key and confirms the action
- **THEN** the key is removed from the list

#### Scenario: Signed-out users have no access

- **WHEN** a signed-out user has no session
- **THEN** the user menu offers no "API keys" entry and the page is not presented
  as an authenticated surface
