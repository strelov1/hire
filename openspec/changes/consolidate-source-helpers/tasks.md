## 1. Remove no-op normalizeNBSP (1.1)

Implementation revealed `normalizeNBSP` is a no-op before `isRemote`: the heuristic
does substring search for "remote"/"удал" (no embedded space), so NBSP→space never
changes the result (proven by probe). It is therefore removed, not folded.

- [x] 1.1 Add a characterization test asserting `isRemote` already flags NBSP-bearing
  remote text (" Удалённо", "Remote work") with no wrapper — green before
  and after, locking the equivalence.
- [x] 1.2 Drop `normalizeNBSP(...)` from all 7 call sites (aviasales, dodo, kuper,
  mtslink, mts, ozon, tbank) so they call `isRemote(...)` directly; delete the
  `normalizeNBSP` function and fix the now-misleading ozon comment. Existing adapter
  tests stay green (behavior-preserving).

## 2. firstNonEmpty helper (1.2)

- [x] 2.1 RED: add a `firstNonEmpty` test (first non-blank wins; all-blank → ""; whitespace-only treated as blank).
- [x] 2.2 GREEN: implement `firstNonEmpty` in `source.go`; replace the inline `x := A; if x == "" { x = B }` fallbacks in dodo/mts/sber (company) and domclick/lamoda (body) where it reads at least as clearly.

## 3. Shared defaultDetailWorkers (1.3)

- [x] 3.1 Replace the 20 per-adapter `xDetailWorkers = 8` constants with one package-level `defaultDetailWorkers = 8` (in `source.go` near `fetchDetails`); update each `fetchDetails(..., xDetailWorkers, ...)` call.

## 4. Move HTML helpers to html.go (1.4)

- [x] 4.1 Move `walk`/`attr`/`textContent`/`itempropHTML` verbatim from `successfactors.go` to a new `internal/sources/html.go` (no signature change); confirm `mts`/`vk`/`successfactors` still compile against them.

## 5. Verify

- [x] 5.1 `go build ./... && go vet ./... && go test ./...` green; `go test ./internal/sources/...` covers the touched adapters.
