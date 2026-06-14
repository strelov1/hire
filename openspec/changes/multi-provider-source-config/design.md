## Context

`internal/sources` loads boards from a per-provider file whose name *is* the provider
(`ParseConfig` stamps every entry with the file-name provider, overwriting any per-entry
value). `cmd/ingest` runs once per file and, after the run, sweeps that one provider's
unseen jobs. `pipeline.Runner.Run` already crawls all entries in one bounded concurrent
pool (size 8) keyed by each entry's provider via the registry — it does not care how many
providers are in the batch. The single-source adapters each get a one-line file + cron
entry only because of the file-name-as-provider convention.

## Goals / Non-Goals

- **Goal:** one `custom.yml` for the small single-source configs; a file may carry several
  providers; the sweep stays correct per provider.
- **Non-Goals:** no change to adapters, `normalizeJob`, the DB, or crawl concurrency
  model; gem/greenhouse-style multi-tenant board lists keep their own files.

## Decisions

- **Provider resolution: per-entry wins, file name is the fallback.** `ParseConfig` no
  longer overwrites a non-empty `entry.Provider`; it only fills the file-name provider
  where the entry left it blank. Existing files (no per-entry provider) behave exactly as
  before; `custom.yml` names a provider on every line. Criterion for `custom.yml`
  membership is "one source" (one company, possibly with host/lang board variants like
  yandex), not "boardless" — `board` is just an optional field that rides along.

- **Validation is per entry.** `Config.Validate` loops entries and checks each against the
  registry by its own resolved provider (registered? boardless-or-has-board?). For
  `custom.yml` the file-name "custom" is not a registered provider, so an entry that omits
  `provider` resolves to "custom" and fails fast — exactly the desired guard.

- **`Runner.Run` returns `map[string]Stats` (per provider).** This is the one signature
  change. The crawl is unchanged (one bounded pool over all entries); stats are tallied
  per `entry.Provider` instead of into a single aggregate. Chosen over having `cmd/ingest`
  group entries by provider and call `Run` once per group, which would nest two bounded
  pools and serialize nothing useful — the flat single-pool crawl already gives the
  desired "vk occupies one slot, the other 12 proceed" behavior.

- **`cmd/ingest` sweeps per provider.** It sums the map for the done-log line, then for
  each provider whose `Stats.Ingested > 0` calls `CloseUnseenJobs(provider)`. A
  single-provider file yields a one-key map — identical behavior to today.

## Risks / Trade-offs

- [Runner.Run signature change ripples to its callers/tests] -> only `cmd/ingest` calls it;
  update it and the pipeline tests. Contained.
- [A provider in custom.yml that totally fails its crawl] -> its `Ingested` is 0, so its
  sweep is skipped (existing guard, now per provider) — no mass-close. Other providers in
  the same run are unaffected (failure already isolated per board in `Run`).
- [vk's 5s/detail pacing in a shared run] -> it holds one of 8 slots; the other providers
  use the remaining 7. Worst case vk is slow, never blocking.

## Migration Plan

Add `sources/custom.yml`, delete the 14 one-line files, update the cron to call
`ingest sources/custom.yml` instead of the 14 per-source lines (ops repo, out of this
change's code). No DB change; rollback is a revert + restoring the cron lines.

## Open Questions

None.
