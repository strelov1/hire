## Context

Remote reach lives only in `enrichment.countries` (ISO 3166-1 alpha-2). Today:

- `jobs.remote BOOLEAN NOT NULL DEFAULT TRUE` — raw flag set by adapters (explicit
  API boolean for Recruitee/SmartRecruiters/Ashby/Workable; `isRemote(location)`
  text heuristic for Greenhouse/Lever). Also fed to the enrichment LLM as a hint
  and surfaced publicly (`jobview.Job.remote`, `?remote=true`, SPA fallback).
- `enrichment.work_mode` — `remote`/`hybrid`/`onsite`.
- `enrichment.countries []string` — country eligibility.

Two cases collapse to empty `countries` and become indistinguishable: **global**
(open anywhere) vs **unknown** (not stated / not yet enriched). Regions (EU…)
can't be expressed at all. Enrichment is async via the `enrichment_outbox` queue,
and everything lives in the `jobs.enrichment` JSONB — no schema migration is
needed to add fields. MVP stage: no persistent data, architecture stays fluid.

## Goals / Non-Goals

**Goals:**
- Make `global`, `regional`, `national` reach first-class and distinguishable
  from `unknown`.
- Back a single curated "Remote type" SPA facet that mixes levels (global /
  region / country).
- Collapse the public notion of "remote" to one source of truth
  (`work_mode` + `remote_scope`), removing the parallel raw flag.

**Non-Goals:**
- Changing the `jobs.remote` column, the source/pipeline `Remote` field, or the
  LLM hint path (all kept).
- A DB migration or `enrich.Version` bump (no persistent data at MVP).
- Reworking the granular `countries` facet (stays) or UI localization (labels
  stay English).
- Hard cross-field consistency enforcement in validation.

## Decisions

### 1. Normalized contract (`remote_scope` + `regions`) over a flat enum

Add to `enrich.Enrichment`: `RemoteScope string` (vocab `global`/`regional`/
`national`) and `Regions []string` (vocab EU, EMEA, EEA, UK, AMERICAS,
NORTH_AMERICA, LATAM, APAC, MENA, AFRICA). `countries` stays for national reach.

| `remote_scope` | `regions` | `countries` | Meaning |
|----------------|-----------|-------------|---------|
| `global`   | empty       | empty        | open anywhere |
| `regional` | non-empty   | usually empty| open within a region |
| `national` | empty       | non-empty    | open in specific countries |
| `""`       | —           | —            | unknown / not enriched |

**Why an explicit `remote_scope` and not derivation:** `global` cannot be
inferred from "no countries" — that is exactly the unknown case. The discriminator
must be stored, set to `global` only on an explicit signal.

*Alternative — single flat `remote_type[]` enum* (global + regions + ISO
countries in one field): fewer fields and a 1:1 facet map, but mixes levels in
one place, complicates validation (region codes vs ISO codes), and partly
duplicates `countries`. Rejected in favor of a clean normalized contract.

### 2. Derived, index-only `remote_type` for the curated facet

The facet panel binds one pill group to one query param, but the mockup's "Remote
type" mixes levels. So the backend exposes a single derived multi-valued field
rather than making the frontend juggle three params.

`search.JobDocument` gains a sibling `RemoteType []string` (NOT inside the
`enrichment` object, keeping the contract clean), computed in `search.FromJob`:
`work_mode != remote` → none; `global` → `["global"]`; `regional` → lowercased
regions; `national` → lowercased countries; else none. Registered filterable
(top-level `remote_type`). It lives only in the index — the regular
`/api/v1/jobs` list/detail responses never carry it, and the SPA reads canonical
`remote_scope`/`regions`/`countries` for display.

Curated facet values are lowercased codes that match the derived field
(`global`, `eu`, `us`, `ru`), extensible without backend change.

### 3. Demote `jobs.remote` to an internal enrichment hint

`jobs.remote` has three roles; only the LLM-hint role is non-redundant:

| Role | Verdict |
|------|---------|
| Public wire `jobview.Job.remote` | dup of `work_mode=remote` → **remove** |
| Search `?remote=true` + filterable `remote` | dup of `work_mode` facet → **remove** |
| SPA `workArrangement()` fallback | dup; splits truth → **remove** |
| LLM input hint (prompt + `enrich.Provider`) | source's explicit boolean, can diverge from location text → **keep** |

The column, the source/pipeline/provider `Remote` field, and the prompt line stay
as the channel that persists the source's remote signal across the async
ingest→enrich boundary. After removal, public "remote" = `work_mode` (+
`remote_scope`) only.

### 4. Soft cross-field validation

`Validate` enforces vocabulary membership for `remote_scope` and `regions`
(out-of-vocab → retry-once → dead-letter, matching existing enum handling) but
does **not** reject cross-field inconsistency (e.g. `regional` with empty
`regions`). Vocabulary violations are objectively wrong; a soft miss is still a
usable payload, and hard-rejecting it would inflate the dead-letter queue.
Consistency is guided by the prompt.

## Risks / Trade-offs

- **Losing the explicit remote boolean as a public/filter signal** → mitigated:
  it is kept as an enrichment input; public filtering moves to `work_mode`, which
  the SPA already exposes.
- **Soft validation lets `regional`-without-`regions` through** → mitigated by
  prompt guidance; acceptable because the payload is still usable and the
  alternative inflates dead-letters. Seam: add a non-fatal normalize pass if soft
  misses prove common.
- **Unenriched jobs show no remote reach** → acceptable and consistent: every
  other enrichment field already behaves this way; the facet simply won't match
  them, same as seniority/category filters.
- **`remote_type` echoed in search responses but unused by the SPA** → harmless;
  display uses the canonical fields, and the field is `omitempty`.

## Migration Plan

No DB migration and no `enrich.Version` bump. Deploy code; the next ingest +
enrich run populates `remote_scope`/`regions` from the updated prompt. Recreate
the dev volume (`docker compose down -v && make up`) if a DB holds stale
enrichments. Rollback = revert the code; the unchanged column and JSONB tolerate
absent fields.

## Open Questions

None — design approved during brainstorming.
