//go:build integration

// Integration test for the company-list endpoint's name search: GET
// /api/v1/companies?q= must filter the returned companies and report the
// filtered count in meta.total (so search pagination is correct). The handler
// uses a concrete *db.Queries, so the wire contract can only be exercised
// against a real Postgres. Run with: go test -tags=integration ./internal/handler/
package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/strelov1/freehire/internal/db"
)

func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	var scripts []string
	for _, f := range []string{
		"0001_init.sql", "0002_companies.sql",
		"0003_job_enrichment.sql", "0004_enrichment_outbox.sql",
		"0005_users.sql", "0006_user_jobs.sql",
		"0007_job_public_slug.sql", "0010_user_identities.sql",
	} {
		abs, err := filepath.Abs(filepath.Join("..", "..", "migrations", f))
		if err != nil {
			t.Fatalf("resolve migration path: %v", err)
		}
		scripts = append(scripts, abs)
	}

	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("hire"),
		postgres.WithUsername("hire"),
		postgres.WithPassword("hire"),
		postgres.WithInitScripts(scripts...),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestListCompaniesSearchEndpoint(t *testing.T) {
	pool := startPostgres(t)
	ctx := context.Background()
	for _, c := range []struct{ slug, name string }{
		{"acme", "Acme Corp"}, {"acme-labs", "ACME Labs"}, {"globex", "Globex"},
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO companies (slug, name) VALUES ($1, $2)`, c.slug, c.name); err != nil {
			t.Fatalf("seed %q: %v", c.slug, err)
		}
	}

	h := &Handler{pool: pool, queries: db.New(pool)}
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Get("/api/v1/companies", h.ListCompanies)

	doList := func(t *testing.T, url string) (names []string, total float64) {
		t.Helper()
		resp, err := app.Test(httptest.NewRequest("GET", url, nil))
		if err != nil {
			t.Fatalf("request %q: %v", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
		}
		var body struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
			Meta struct {
				Total float64 `json:"total"`
			} `json:"meta"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, c := range body.Data {
			names = append(names, c.Name)
		}
		return names, body.Meta.Total
	}

	t.Run("q filters companies and meta.total is the filtered count", func(t *testing.T) {
		names, total := doList(t, "/api/v1/companies?q=acme")
		if len(names) != 2 {
			t.Errorf("names = %v, want 2 ACME companies", names)
		}
		for _, n := range names {
			if !strings.Contains(strings.ToLower(n), "acme") {
				t.Errorf("returned non-matching company %q for q=acme", n)
			}
		}
		if total != 2 {
			t.Errorf("meta.total = %v, want 2 (filtered count)", total)
		}
	})

	t.Run("empty q returns the full list", func(t *testing.T) {
		names, total := doList(t, "/api/v1/companies")
		if len(names) != 3 || total != 3 {
			t.Errorf("full list: names=%v total=%v, want 3/3", names, total)
		}
	})
}
