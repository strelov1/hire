## 1. Backend CORS

- [x] 1.1 Add `github.com/gofiber/fiber/v2/middleware/cors` to `go.mod` (`go get`)
- [x] 1.2 Add an allowed-origin field to `internal/config/config.go` (env, e.g. `FRONTEND_ORIGIN`, default `http://localhost:5173`)
- [x] 1.3 Wire the CORS middleware in `internal/handler/handler.go` `Register`, using the configured origin
- [x] 1.4 Verify `go build ./... && go vet ./...` pass and a manual `OPTIONS`/`GET` to `/api/v1/jobs` returns `Access-Control-Allow-Origin`

## 2. Frontend scaffold

- [x] 2.1 Create `web/` with `package.json` (Svelte 5, Vite 6, TypeScript, Tailwind v4, shadcn-svelte/bits-ui, tailwind-variants, clsx, tailwind-merge, @lucide/svelte, tw-animate-css)
- [x] 2.2 Add `vite.config.ts`, `svelte.config.js`, `tsconfig.json`, `index.html`, `.env.example` (`VITE_API_URL`)
- [x] 2.3 Add `components.json` and Tailwind setup so shadcn-svelte components can be added under `src/lib/ui/`
- [x] 2.4 Create `src/app.css` with the oklch palette and `.dark` variant; create `src/main.ts` that calls `initTheme()` and mounts `App`
- [x] 2.5 Verify `npm install` and `npm run dev` start the dev server with a blank shell

## 3. Core libs

- [x] 3.1 Create `src/lib/utils.ts` with `cn()` (clsx + tailwind-merge)
- [x] 3.2 Create `src/lib/types.ts` with `Job`, `Company`, `ListMeta` mirroring `internal/db/models.go` and query columns
- [x] 3.3 Create `src/lib/api.ts` with `listJobs`, `getJob`, `listCompanies`, `getCompany` over `fetch` using `VITE_API_URL`
- [x] 3.4 Create `src/lib/theme.svelte.ts` (light/dark/system, localStorage, `.dark` class, `initTheme()`)
- [x] 3.5 Create `src/lib/router.svelte.ts` (History API rune router mapping path → view + params)

## 4. Shell and shared components

- [x] 4.1 Add shadcn-svelte primitives used by views (button, badge, card, skeleton) under `src/lib/ui/`
- [x] 4.2 Create `src/lib/components/ThemeToggle.svelte`
- [x] 4.3 Create `src/lib/components/TopBar.svelte` (logo + Jobs/Companies nav + ThemeToggle)
- [x] 4.4 Create `src/lib/components/States.svelte` (Loading / Empty / Error)
- [x] 4.5 Create `src/lib/components/JobRow.svelte` (reusable job row linking to job detail)
- [x] 4.6 Create `src/App.svelte` wiring TopBar + router to the current view

## 5. Views

- [x] 5.1 Create `src/lib/components/JobsView.svelte` (jobs list + limit/offset pagination via `meta.total`, States)
- [x] 5.2 Create `src/lib/components/JobView.svelte` (job detail, company link, badges, Apply link, 404 → error state)
- [x] 5.3 Create `src/lib/components/CompaniesView.svelte` (companies list with job count, States)
- [x] 5.4 Create `src/lib/components/CompanyView.svelte` (company info + its jobs via JobRow, States)

## 6. Verification

- [x] 6.1 Run `npm run check` (svelte-check) and fix type errors
- [x] 6.2 Run backend locally + `npm run dev`; manually confirm every route loads, paginates, toggles theme, and degrades to Empty/Error states
- [x] 6.3 Run `npm run build` and confirm a clean static build
