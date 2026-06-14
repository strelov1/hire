## Why

The `internal/sources` package has grown to ~35 adapters, and a few small pieces of
logic have been copy-pasted across them as the set grew: a "first non-empty"
fallback, the detail-fan-out worker count, and a set of generic HTML helpers
stranded inside one adapter file. A fourth duplication, a non-breaking-space fixup
before the remote heuristic, turned out to be a no-op. Each copy is a place behavior
can silently drift. Folding the live ones into the shared layer (and deleting the
dead one) keeps adapters thin and the shared heuristics single-sourced.

## What Changes

- Remove `normalizeNBSP` and its 7 call sites: it is a proven no-op before the remote
  heuristic, which substring-matches "remote"/"udal" (neither contains a space), so
  the NBSP-to-space pass never changes the result. Call sites collapse to plain
  `isRemote(...)`. Dead code, removed.
- Add a `firstNonEmpty(parts ...string) string` helper alongside `joinNonEmpty` and
  use it for the `company`/body fallbacks currently written inline as
  `x := A; if x == "" { x = B }` (dodo, mts, sber, and the body fallbacks in
  domclick/lamoda where it reads cleanly).
- Replace the 20 per-adapter `xDetailWorkers = 8` constants with a single
  package-level `defaultDetailWorkers = 8`.
- Move the generic HTML helpers `walk`/`attr`/`textContent`/`itempropHTML` out of
  `successfactors.go` into a new `internal/sources/html.go` (pure relocation, no
  signature change) so adapters that use them (mts, vk) no longer reach into a
  sibling adapter's file.

No behavior change: this is an internal refactor. Output of every adapter is
byte-identical for the same input.

## Capabilities

### New Capabilities
<!-- none — no new behavior -->

### Modified Capabilities
<!-- none — requirements of source-ingest are unchanged; this is implementation-only -->

## Impact

- Code only, all within `internal/sources`: `source.go`, the remote heuristic,
  `successfactors.go`, and the adapters that drop `normalizeNBSP`/inline
  fallbacks/local worker constants (aviasales, dodo, kuper, mtslink, mts, ozon,
  tbank, sber, domclick, lamoda, and the ~20 detail adapters for the worker const).
- New file `internal/sources/html.go`; `normalizeNBSP` removed.
- No spec delta, no API change, no migration, no config change.
- Verification: `go build ./... && go vet ./... && go test ./...` stays green.
