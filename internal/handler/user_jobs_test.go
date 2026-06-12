package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/auth"
)

// userJobsApp mounts the view/apply routes behind RequireAuth on a handler with
// no DB. The auth-gate cases below reject before any query runs, so the nil
// queries is never dereferenced. Slug resolution and the DB-backed happy path /
// idempotency are covered by the db-package integration tests (GetJobBySlug,
// TestUserJobs); an unknown slug surfaces as pgx.ErrNoRows → 404 via ErrorHandler.
func userJobsApp() (*fiber.App, *auth.Issuer) {
	iss := auth.NewIssuer("test-secret", time.Hour)
	h := &Handler{issuer: iss}
	app := fiber.New()
	app.Post("/jobs/:slug/view", auth.RequireAuth(iss), h.RecordView)
	app.Post("/jobs/:slug/apply", auth.RequireAuth(iss), h.MarkApplied)
	app.Post("/jobs/:slug/save", auth.RequireAuth(iss), h.SaveJob)
	app.Delete("/jobs/:slug/save", auth.RequireAuth(iss), h.UnsaveJob)
	return app, iss
}

func postUserJob(t *testing.T, app *fiber.App, path, token string) int {
	t.Helper()
	return requestUserJob(t, app, fiber.MethodPost, path, token)
}

func requestUserJob(t *testing.T, app *fiber.App, method, path, token string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	return resp.StatusCode
}

func TestRecordView_RequiresAuth(t *testing.T) {
	app, _ := userJobsApp()
	if got := postUserJob(t, app, "/jobs/go-dev-acme-t35nijto/view", ""); got != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", got)
	}
}

func TestMarkApplied_RequiresAuth(t *testing.T) {
	app, _ := userJobsApp()
	if got := postUserJob(t, app, "/jobs/go-dev-acme-t35nijto/apply", ""); got != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", got)
	}
}

func TestSaveJob_RequiresAuth(t *testing.T) {
	app, _ := userJobsApp()
	if got := postUserJob(t, app, "/jobs/go-dev-acme-t35nijto/save", ""); got != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", got)
	}
}

func TestUnsaveJob_RequiresAuth(t *testing.T) {
	app, _ := userJobsApp()
	if got := requestUserJob(t, app, fiber.MethodDelete, "/jobs/go-dev-acme-t35nijto/save", ""); got != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", got)
	}
}

// interactionResponse is the only interaction shape that reaches a response. This
// locks the contract: it omits user_id and carries job_id + the three timestamps.
func TestInteractionResponse_Shape(t *testing.T) {
	raw, err := json.Marshal(interactionResponse{JobID: 7})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, leaked := fields["user_id"]; leaked {
		t.Error("interactionResponse must not include user_id")
	}
	for _, want := range []string{"job_id", "viewed_at", "saved_at", "applied_at"} {
		if _, ok := fields[want]; !ok {
			t.Errorf("interactionResponse missing %q", want)
		}
	}
}
