# Web Frontend Design

Status: approved
Date: 2026-06-07

## Goal

A minimal web frontend for the `hire` job aggregator that browses the existing
read-only HTTP API: jobs and companies, list and detail views. Visual language
is a flat, low-chrome design with light and dark themes and deliberately simple
elements.

## Scope

In scope (first version):

- Jobs list with limit/offset pagination
- Job detail
- Companies list (with per-company job count)
- Company detail (company info + its jobs)
- Light / dark / system theme

Out of scope (YAGNI — not built now):

- Authentication
- Server-side search/filtering (the API only supports limit/offset pagination)
- SSR
- Any write path (the API is read-only)

## Stack

- **Svelte 5** (runes) + **Vite 6** + **TypeScript**
- **Tailwind CSS v4** via `@tailwindcss/vite`
- **shadcn-svelte** components on **bits-ui** + **tailwind-variants** +
  `clsx`/`tailwind-merge` (exposed through a `cn()` utility)
- **@lucide/svelte** for icons, `tw-animate-css` for transitions
- Client routing: a small rune-based router over the History API
  (`src/lib/router.svelte.ts`, ~40 lines). No framework router is needed for
  four read-only screens.

## Layout

```
web/
  index.html
  package.json  vite.config.ts  svelte.config.js  tsconfig.json  components.json
  src/
    main.ts            entry point: initTheme() then mount(App)
    app.css            Tailwind import + oklch palette + .dark variant
    App.svelte         shell: TopBar + <Router> rendering the current view
    lib/
      api.ts           typed fetch client (VITE_API_URL)
      types.ts         Job, Company, ListMeta
      router.svelte.ts rune router (path -> view + params)
      theme.svelte.ts  theme controller (light/dark/system, localStorage)
      utils.ts         cn() and small helpers
      ui/              shadcn-svelte components (button, badge, card, skeleton, ...)
      components/
        TopBar.svelte        logo + nav (Jobs/Companies) + ThemeToggle
        ThemeToggle.svelte
        JobsView.svelte      jobs list + pagination
        JobView.svelte       job detail
        CompaniesView.svelte companies list (with count)
        CompanyView.svelte   company + its jobs
        JobRow.svelte        reusable job row
        States.svelte        Loading / Empty / Error
```

## Routes and screens

| Route             | View          | Contents |
|-------------------|---------------|----------|
| `/`               | JobsView      | Rows of jobs (title, company, location, remote badge, source, date). limit/offset pagination via "Load more" / page controls, driven by `meta.total`. |
| `/jobs/:id`       | JobView       | Title, link to company, badges (remote/source), date, description (plain text), "Apply" link to `job.url`. |
| `/companies`      | CompaniesView | Rows of companies with job count. |
| `/companies/:slug`| CompanyView   | Company info + its jobs (reusing `JobRow`). |

## Data layer

`api.ts` exposes `listJobs`, `getJob`, `listCompanies`, `getCompany` wrapping
`fetch`. Base URL: `import.meta.env.VITE_API_URL ?? 'http://localhost:8080'`.
Response shapes mirror the backend: lists return `{ data, meta }`, single items
return `{ data }`. Each view loads via runes (`$state` + an `$effect` keyed on
route params) and renders Loading / Empty / Error states.

Types in `types.ts` mirror the backend models:

- `Job`: `id, source, external_id, url, title, company, company_slug?, location,
  remote, description, posted_at, created_at, updated_at`
- `Company`: slug-keyed fields + `job_count`
- `ListMeta`: `{ total, limit, offset }`

(Exact field names are confirmed against `internal/db/models.go` and the query
result columns during implementation.)

## Theme and design language

Flat `oklch` palette with light and dark themes. Dark mode toggles a `.dark`
class on `<html>`; the choice persists in localStorage, and a `system` mode
tracks `prefers-color-scheme`. A single background fills the whole canvas;
separation comes from spacing and thin borders rather than competing fills —
keeping elements as simple as possible. `theme.svelte.ts` owns the controller;
`main.ts` calls `initTheme()` once at boot.

## Backend change

The frontend calls the API directly from the browser on a different origin
(`:5173` dev, or wherever the static build is served). Add CORS middleware
(`github.com/gofiber/fiber/v2/middleware/cors`) in `handler.Register` (or
`main.go`) allowing the frontend origin. This is the only Go change required.

## Component boundaries

- `api.ts` — the only place that knows about HTTP and the API base URL.
  Views depend on its typed functions, not on `fetch`.
- `router.svelte.ts` — owns URL <-> (view, params) mapping. Views read params,
  never parse the URL themselves.
- `theme.svelte.ts` — owns theme state and the `.dark` class. Components read
  `themeStore` and call `setMode`.
- `JobRow.svelte` — single source of truth for how a job appears in any list
  (JobsView and CompanyView both use it).
- `States.svelte` — shared Loading/Empty/Error rendering so every view handles
  the three async states the same way.

## Testing

No automated test suite exists in the repo yet, and none is added here. Manual
verification: run the backend locally, run `vite`, and confirm each route loads,
paginates, toggles theme, and degrades to Empty/Error states correctly.
