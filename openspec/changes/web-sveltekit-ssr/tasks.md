## 1. SvelteKit scaffold & shared client

- [x] 1.1 Add SvelteKit + `adapter-node` to `web/` (deps, `svelte.config.js`,
  `vite.config.ts`, `app.html`, `src/app.d.ts`); keep `app.css`, `lib/`
  components, `types.ts`. `svelte-check` passes on the empty skeleton.
- [x] 1.2 Refactor `lib/api.ts` to accept an injected `fetch` so the same client
  works in server `load` and in the browser; keep the existing function shapes.
- [x] 1.3 Wire the dev server to proxy `/api` (and `/health`) to the Go backend
  so dev stays same-origin (replaces the Vite proxy); a dev request to `/api/v1/jobs`
  reaches the backend.

## 2. Public detail routes (SSR) — Slice A

Detail pages load by slug and have no dependency on the filter/search state, so
they port cleanly ahead of the lists. Interactive bits (record-view, save,
apply) hydrate client-side over the server-rendered article.

- [x] 2.1 `src/routes/jobs/[slug]/+page.svelte` + `+page.server.ts`: job detail
  server-rendered from the public job API via `event.fetch`; a 404 yields an
  error page with a not-found status; closed-job state and Apply link preserved;
  save/apply/record-view hydrate client-side, gated on auth.
- [x] 2.2 `src/routes/companies/[slug]/+page.svelte` + `+page.server.ts`: company
  detail server-rendered, reusing the job-row presentation.

## 3. List routes + interactive state (SSR) — Slice B

The `/jobs` list and `/companies` list are inseparable from the migration of
their URL-synced state (filters/search) off the hand-rolled `router` and onto
SvelteKit primitives, so each list ships together with its state migration. The
pure `filtersFromParams`/`filtersToParams` helpers are reused unchanged in
`load`; only the `FilterStore`/`router` coupling is reworked.

- [x] 3.1 Port the filter/search URL-sync off `router` onto SvelteKit
  (`page.url.searchParams` + `goto`/`replaceState`), preserving the synchronous
  write-state→URL-in-handler pattern (no controlled-input revert).
- [x] 3.2 `src/routes/jobs/+page.svelte` + `+page.server.ts`: jobs list (`/jobs`),
  first page server-rendered from the search API (`filtersFromParams(url)` →
  seed the `Paginator`); rows in initial HTML; "load more", filters, sort, and
  reach indicators preserved.
- [x] 3.3 `src/routes/companies/+page.svelte` + `load`: companies list
  server-rendered, with the debounced name search URL-synced (`?q=`) preserved.
- [x] 3.4 `src/routes/+page.svelte` (home `/`): server-render the marketing
  landing.

## 4. Layout, auth, theme, my/* surfaces

- [x] 4.1 Root `+layout.svelte` + `+layout.server.ts`: resolve the current user
  from `/me` (cookie forwarded) so signed-in chrome renders server-side without a
  post-mount flash; client hydrates from layout data; TopBar/footer chrome.
  **Must re-wire `initAuth()`** — it lived in the deleted `main.ts`, so until the
  layout calls it (or the server resolves the user), all per-user interactions
  (Save, Apply prompt, record-view, "You applied") are inert. The server-resolve
  approach supersedes a bare client `initAuth`.
- [x] 4.2 Theme: persist the choice in a cookie so SSR sets the `.dark` class on
  `<html>`; keep the system-mode inline fallback in `app.html`; no theme FOUC.
- [x] 4.3 Port `/my/jobs` and `/my/api-keys` routes (auth-guarded); retire
  `App.svelte`, `router.svelte.ts`, and the api.ts compatibility re-exports.

## 5. SEO artifacts

- [x] 5.1 Per-route `<head>`: job-specific `<title>`/description/canonical/OG on
  `/jobs/:slug`; page-appropriate metadata on the list pages.
- [x] 5.2 `JobPosting` JSON-LD builder (unit-testable, from the job-view shape) +
  `Organization` JSON-LD on company pages; emitted server-side; closed jobs
  reflect their status.
- [x] 5.3 `src/routes/robots.txt/+server.ts`: real `text/plain` robots file
  referencing the sitemap.
- [x] 5.4 `src/routes/sitemap.xml/+server.ts`: generated XML of job and company
  URLs (decide source: existing listing vs. a minimal slug+timestamp query; if
  capped, `log` what is omitted).

## 6. Deploy topology

- [x] 6.1 New `web/Dockerfile` running the `adapter-node` server; update
  `web/nginx.conf` so nginx fronts `/api`+`/health`→Go and everything else→Node;
  verify the full stack in Docker (`make up`) serves SSR pages. **Set `ORIGIN`**
  (public site URL) and **`API_INTERNAL_URL`** (Go service origin) on the web
  service, and forward Host/X-Forwarded-* in nginx — otherwise canonical/og:url/
  JSON-LD/sitemap URLs carry the internal Node origin (see group-5 review).

## 7. Verification

- [x] 7.1 Verify each spec scenario: `curl` `/jobs/:slug`, `/jobs`, `/companies`
  return content-bearing HTML; `JobPosting` JSON-LD validates; `/robots.txt` is
  text/plain; `/sitemap.xml` is valid XML; signed-in SSR renders correct chrome;
  no hydration warnings; `svelte-check` clean.
