## 1. Per-entry provider resolution + validation

- [ ] 1.1 RED: in `config_test.go`, add cases — an entry with its own `provider` keeps it
  (file-name does not overwrite); an entry without `provider` falls back to the file name;
  a mixed file yields entries with different providers.
- [ ] 1.2 GREEN: `ParseConfig` fills the file-name provider only where `entry.Provider`
  is empty (stop overwriting non-empty values).
- [ ] 1.3 RED: add `Validate` cases — a registered per-entry provider passes; an entry
  resolving to an unregistered provider (e.g. file name "custom" with no per-entry
  provider) fails fast; boardless/empty-board check is applied per entry's own provider.
- [ ] 1.4 GREEN: `Config.Validate` loops entries, validating each by its resolved provider.

## 2. Runner.Run returns per-provider stats

- [ ] 2.1 RED: in `pipeline_test.go`, assert `Run` returns `map[string]Stats` keyed by
  provider (multi-provider batch → one key per provider; per-provider Ingested/Failed
  correct; failure isolation preserved).
- [ ] 2.2 GREEN: change `Run` to tally `map[string]Stats` keyed by `entry.Provider`; keep
  the single bounded concurrent pool. Update existing pipeline tests to the new shape.

## 3. cmd/ingest per-provider sweep

- [ ] 3.1 RED: extend the cmd/ingest test so a run with two providers (one ingested>0, one
  ingested=0) sweeps only the first; the done-log aggregates the map.
- [ ] 3.2 GREEN: `cmd/ingest` consumes `map[string]Stats`: logs the summed totals, then
  for each provider with `Ingested > 0` calls `CloseUnseenJobs(provider)`.

## 4. custom.yml + remove one-line files

- [ ] 4.1 Add `sources/custom.yml` with the 13 boardless single-source providers plus
  yandex's two `board: ru`/`board: com` entries, each naming its `provider`. Delete the 14
  one-line files (vk, ozon, sber, alfabank, lamoda, kuper, aviasales, dodo, domclick,
  mtslink, tbank, mts, rwb, yandex). Keep gem.yml.
- [ ] 4.2 Validate `custom.yml` loads + passes `Validate` against the real registry (a
  config_test over the file, or a quick `go run ./cmd/ingest sources/custom.yml` dry check
  documented — no DB needed for the validate path which fails fast before connecting).

## 5. Verify

- [ ] 5.1 `go build ./... && go vet ./... && go test ./...` green.
