## Context

The `hire` backend is a Fiber HTTP server exposing read-only endpoints for jobs
and companies (`{data, meta}` for lists, `{data}` for single items). There is no
client. This change adds a browser SPA under `web/` and the one backend change
needed for the browser to reach the API directly: CORS.

Constraints: the API supports only `limit`/`offset` pagination (no server-side
search or filtering), is read-only, and has no auth. The frontend mirrors those
limits rather than inventing capabilities the backend lacks.

## Goals / Non-Goals

**Goals:**

- A usable SPA covering the full current API: jobs and companies, list + detail.
- Flat, low-chrome visual language with light/dark/system theming.
- Minimal, idiomatic stack; static build deployable behind any static host.
- A single, surgical backend change (CORS) to enable direct browser access.

**Non-Goals:**

- Authentication, write paths, SSR.
- Server-side search/filtering (not offered by the API).
- An automated frontend test suite (the repo has none yet; out of scope here).

## Decisions

**Stack: Svelte 5 (runes) + Vite 6 + TypeScript + Tailwind v4 + shadcn-svelte.**
shadcn-svelte (on `bits-ui`) plus `tailwind-variants` and a `cn()` helper give
accessible primitives (button, badge, card, skeleton) without hand-rolling them.
Alternatives: a dependency-free hand-rolled component set (lighter but more code
to own); SvelteKit (file routing + SSR, but changes the deploy model to a Node
server — unnecessary for a static SPA over a Go API). Rejected both: the UI-kit
SPA balances ready-made components against a simple static deploy.

**Routing: a small rune-based router over the History API** in
`src/lib/router.svelte.ts`. Four read-only screens don't justify a framework
router. The router owns URL ↔ (view, params) mapping; views read params and
never parse the URL themselves.

**Data access isolated in `src/lib/api.ts`.** It is the only module that knows
the API base URL (`import.meta.env.VITE_API_URL ?? 'http://localhost:8080'`) and
the wire shapes. Views call typed functions (`listJobs`, `getJob`,
`listCompanies`, `getCompany`); types live in `src/lib/types.ts` mirroring the
backend models.

**Theme controller in `src/lib/theme.svelte.ts`.** Owns theme state and the
`.dark` class; `main.ts` calls `initTheme()` once at boot; components read
`themeStore` and call `setMode`. Matches the existing project pattern of small
rune store modules.

**CORS via Fiber middleware** (`github.com/gofiber/fiber/v2/middleware/cors`)
wired in `handler.Register`. The allowed origin comes from config so dev and
production can differ; default permits the Vite dev origin.

**Shared presentation units.** `JobRow.svelte` is the single source of truth for
how a job appears in any list (jobs list and company detail). `States.svelte`
centralizes loading/empty/error rendering so every view handles the three async
states identically.

## Risks / Trade-offs

- **Client-only pagination, no search** → Acceptable: mirrors the API. When the
  backend gains filters, the `api.ts` seam absorbs them without view rewrites.
- **CORS misconfiguration could over-expose the API** → Mitigate by sourcing the
  allowed origin from config rather than wildcarding in production.
- **shadcn-svelte/bits-ui adds dependency weight** → Accepted per the chosen
  approach; offset by not hand-maintaining accessible primitives.
- **Field-name drift between frontend types and backend models** → Mitigate by
  confirming names against `internal/db/models.go` and query result columns
  during implementation.
