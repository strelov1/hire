## 1. Search default ordering

- [x] 1.1 Failing handler test: empty `q` + no `sort` → search called with `created_at:desc`; non-empty `q` + no `sort` → nil sort (relevance); explicit `?sort=created_at&order=asc` honored
- [x] 1.2 Implement: `created_at` in `searchSortable`, default in `searchSort` (gated on empty q), `created_at` in index SortableAttributes; green

## 2. DB list ordering

- [x] 2.1 ListJobs orders by `created_at DESC, id DESC`; regenerate sqlc; adjust/extend integration test if one asserts order

## 3. Verify

- [x] 3.1 build/vet/test green; reindex against dev Meili applies new sortable settings; SPA list shows newest-added first
