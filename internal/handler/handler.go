package handler

import (
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/strelov1/hire/internal/auth"
	"github.com/strelov1/hire/internal/db"
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
}

// pageParams reads and clamps the shared limit/offset pagination query params.
// The offset is clamped into int32 range because the column binds as a Postgres
// int4, and an unbounded query value would otherwise overflow on the conversion.
func pageParams(c *fiber.Ctx) (limit, offset int) {
	limit = min(max(c.QueryInt("limit", defaultLimit), 1), maxLimit)
	offset = min(max(c.QueryInt("offset", 0), 0), math.MaxInt32)
	return limit, offset
}

// Register wires all routes onto the application. frontendOrigin is the single
// browser origin allowed to call the API cross-origin; jwtSecret/jwtTTL
// configure the token issuer behind the auth endpoints; cookieSecure marks the
// auth cookie HTTPS-only. Auth is same-origin only: the SPA reaches the API
// under one origin (a dev Vite proxy mirrors the production reverse proxy), so
// the cookie rides along with no CORS. The CORS allowlist is not credentialed —
// it only permits non-credentialed cross-origin reads of the public endpoints.
func Register(app *fiber.App, pool *pgxpool.Pool, frontendOrigin, jwtSecret string, jwtTTL time.Duration, cookieSecure bool) {
	h := &Handler{
		pool:         pool,
		queries:      db.New(pool),
		issuer:       auth.NewIssuer(jwtSecret, jwtTTL),
		cookieSecure: cookieSecure,
	}

	app.Use(cors.New(cors.Config{AllowOrigins: frontendOrigin}))

	app.Get("/health", h.Health)

	api := app.Group("/api/v1")
	api.Get("/jobs", h.ListJobs)
	api.Get("/jobs/:id", h.GetJob)
	api.Get("/companies", h.ListCompanies)
	api.Get("/companies/:slug", h.GetCompany)

	// Auth: register/login/logout are public (logout just clears the cookie);
	// me is guarded by the auth-cookie check.
	authGroup := api.Group("/auth")
	authGroup.Post("/register", h.Register)
	authGroup.Post("/login", h.Login)
	authGroup.Post("/logout", h.Logout)
	authGroup.Get("/me", auth.RequireAuth(h.issuer), h.Me)
}
