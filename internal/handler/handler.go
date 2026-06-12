package handler

import (
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/strelov1/freehire/internal/auth"
	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/search"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Handler holds dependencies shared across HTTP handlers.
type Handler struct {
	pool         *pgxpool.Pool
	queries      *db.Queries
	issuer       *auth.Issuer
	cookieSecure bool
	// search is the job-search backend. Nil when Meilisearch is unconfigured —
	// the search endpoint then reports 503 and the rest of the API is unaffected.
	search searcher
}

// pageParams reads and clamps the shared limit/offset pagination query params.
// The offset is clamped into int32 range because the column binds as a Postgres
// int4, and an unbounded query value would otherwise overflow on the conversion.
func pageParams(c *fiber.Ctx) (limit, offset int) {
	limit = min(max(c.QueryInt("limit", defaultLimit), 1), maxLimit)
	offset = min(max(c.QueryInt("offset", 0), 0), math.MaxInt32)
	return limit, offset
}

// listResponse writes the shared paginated-list envelope: the data slice plus a
// meta block carrying the filtered total and the limit/offset echoed back. It is
// the single source of the list wire shape, so the jobs/companies/search list
// endpoints cannot drift from one another.
func listResponse(c *fiber.Ctx, data any, total int64, limit, offset int) error {
	return c.JSON(fiber.Map{
		"data": data,
		"meta": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// Register wires all routes onto the application. frontendOrigin is the single
// browser origin allowed to call the API cross-origin; jwtSecret/jwtTTL
// configure the token issuer behind the auth endpoints; cookieSecure marks the
// auth cookie HTTPS-only. Auth is same-origin only: the SPA reaches the API
// under one origin (a dev Vite proxy mirrors the production reverse proxy), so
// the cookie rides along with no CORS. The CORS allowlist is not credentialed —
// it only permits non-credentialed cross-origin reads of the public endpoints.
func Register(app *fiber.App, pool *pgxpool.Pool, frontendOrigin, jwtSecret string, jwtTTL time.Duration, cookieSecure bool, searchClient *search.Client) {
	h := &Handler{
		pool:         pool,
		queries:      db.New(pool),
		issuer:       auth.NewIssuer(jwtSecret, jwtTTL),
		cookieSecure: cookieSecure,
	}
	// Assign only when configured: a nil *search.Client wrapped in the searcher
	// interface would be a non-nil interface and defeat the nil check.
	if searchClient != nil {
		h.search = searchClient
	}

	app.Use(cors.New(cors.Config{AllowOrigins: frontendOrigin}))

	app.Get("/health", h.Health)

	api := app.Group("/api/v1")
	api.Get("/jobs", h.ListJobs)
	// Literal route before the :slug param route so "search" is not read as a slug.
	api.Get("/jobs/search", h.SearchJobs)
	api.Get("/jobs/:slug", h.GetJob)
	api.Get("/companies", h.ListCompanies)
	api.Get("/companies/:slug", h.GetCompany)

	// Per-user job interactions: recording a view and marking applied both
	// require a session. The public job reads above stay unauthenticated. Jobs
	// are addressed by their public slug; the handlers resolve it to the internal
	// id before writing user_jobs.
	api.Post("/jobs/:slug/view", auth.RequireAuth(h.issuer), h.RecordView)
	api.Post("/jobs/:slug/apply", auth.RequireAuth(h.issuer), h.MarkApplied)

	// Auth: register/login/logout are public (logout just clears the cookie);
	// me is guarded by the auth-cookie check.
	authGroup := api.Group("/auth")
	authGroup.Post("/register", h.Register)
	authGroup.Post("/login", h.Login)
	authGroup.Post("/logout", h.Logout)
	authGroup.Get("/me", auth.RequireAuth(h.issuer), h.Me)
}
