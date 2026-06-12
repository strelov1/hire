## Why

A remote job's *geographic reach* matters — open worldwide, restricted to a
region (EU), or to specific countries (US) — but the model can't express it. Reach
is captured only by `enrichment.countries` (ISO codes), so "open anywhere"
(global) and "reach not stated / not yet enriched" (unknown) both collapse to an
empty `countries` array and become indistinguishable, and regions like EU have no
representation at all (they are not ISO countries). Meanwhile the raw
`jobs.remote` boolean duplicates `work_mode=remote` as a parallel public concept,
forcing flag-vs-enrichment reconciliation.

## What Changes

- Add an explicit **remote-scope** dimension to the enrichment contract:
  `remote_scope` (`global` | `regional` | `national`; empty = unknown) and
  `regions` (controlled vocabulary: EU, EMEA, EEA, UK, AMERICAS, NORTH_AMERICA,
  LATAM, APAC, MENA, AFRICA). `remote_scope` is the explicit discriminator that
  makes **global ≠ unknown** — it cannot be derived from the presence/absence of
  countries. Both fields are meaningful only when `work_mode = remote`.
- Validate the new enum fields against their vocabularies (reject out-of-vocab →
  dead-letter, as today). Cross-field consistency (e.g. `regional` ⇒ `regions`
  non-empty) is guided by the prompt, **not** hard-enforced, to avoid inflating
  the dead-letter queue.
- Extend the LLM extraction prompt to populate `remote_scope`/`regions`
  (`global` only on an explicit "worldwide/anywhere").
- Expose a derived, index-only **`remote_type`** multi-valued field on the search
  document (computed from `remote_scope`/`regions`/`countries`, gated on
  `work_mode=remote`) so a single curated facet can mix levels. Register it
  filterable; wire a `remote_type` search-filter param.
- SPA: add a curated **"Remote type"** pills facet (Global / Russia / Europe /
  USA, excludable) under "Work format", and display a remote job's reach
  explicitly (Global / region / countries) instead of showing nothing for global.
- **BREAKING (internal, pre-launch):** demote `jobs.remote` from a public field
  to an internal enrichment hint. Remove it from the public job wire
  (`jobview.Job.remote`), from the search filter (`?remote=true`) and filterable
  attributes, and from the SPA fallback. The **column stays**, populated by every
  adapter and fed to the LLM as the source's explicit remote signal. Public
  "remote" is henceforth solely `work_mode` (+ `remote_scope`).
- No `enrich.Version` bump and no DB migration: at MVP there is no persistent
  data, so fresh ingest + enrich fills the new fields; recreate the dev volume if
  needed.

## Capabilities

### New Capabilities
<!-- None. The change extends existing enrichment, search, and frontend
     capabilities rather than introducing a new one. -->

### Modified Capabilities
- `job-enrichment`: the typed enrichment contract gains the `remote_scope` enum
  and `regions` enum-array (with vocabularies and validation); the public jobs
  read API drops the now-redundant raw `remote` field (subsumed by `work_mode`).
- `job-search`: the index filterable attributes drop `remote` and add the derived
  `remote_type`; the search endpoint replaces the `remote` filter with
  `remote_type`.
- `web-frontend`: list and detail views express a job's remote status (and
  global/regional/national reach) from `work_mode`/`remote_scope` rather than the
  raw flag; the filter UI gains a curated "Remote type" facet.

## Impact

- **Code (enrichment):** `internal/enrich/enrichment.go` (fields, vocabularies,
  `Validate`), `internal/enrich/langchain.go` (prompt). No new dependency.
- **Code (search):** `internal/search/document.go` (derived `remote_type`),
  `internal/search/client.go` (filterable attrs), `internal/handler/search.go`
  (param mapping; remove `remote`).
- **Code (wire):** `internal/jobview/jobview.go` (add scope/regions via embedded
  Enrichment; remove `remote`); regenerate nothing (sqlc unaffected — column
  stays).
- **Code (frontend):** `web/src/lib/types.ts`, `web/src/lib/facets.ts`,
  `web/src/lib/enrichment.ts`, `JobView.svelte`/`JobRow.svelte`.
- **Kept untouched:** `jobs.remote` column, `sources/*` + `pipeline` `Remote`
  field, `enrich.Provider` input, `cmd/ingest`/`cmd/enrich` — the remote hint
  path. No migration, no `enrich.Version` change.
- **Tests:** `Enrichment.Validate` cases; `remote_type` derivation; search
  integration + jobview literals lose `remote`; frontend display.
