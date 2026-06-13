package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/auth"
)

// trackApp mounts the track route on a handler with no DB. The auth gate and the
// body validation (empty / unknown stage) all reject before any query runs, so
// the nil queries is never dereferenced. The DB-backed path is covered by the
// user_jobs integration tests.
func trackApp() (*fiber.App, *auth.Issuer) {
	iss := auth.NewIssuer("test-secret", time.Hour)
	h := &Handler{issuer: iss}
	app := fiber.New()
	app.Patch("/jobs/:slug/track", auth.RequireAuth(iss), h.TrackJob)
	return app, iss
}

func TestTrackJob_RejectsEmptyAndUnknownStage(t *testing.T) {
	app, iss := trackApp()
	token, _ := iss.Issue(7)
	cases := []struct {
		name, body string
	}{
		{"empty body", `{}`},
		{"unknown stage", `{"stage":"banana"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(fiber.MethodPatch, "/jobs/go-dev/track", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Test: %v", err)
			}
			if resp.StatusCode != fiber.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestTrackJob_RequiresAuth(t *testing.T) {
	app, _ := trackApp()
	req := httptest.NewRequest(fiber.MethodPatch, "/jobs/go-dev/track", strings.NewReader(`{"stage":"interview"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIsValidStage(t *testing.T) {
	for _, s := range []string{"applied", "screening", "responded", "interview", "offer", "accepted", "rejected", "withdrawn"} {
		if !isValidStage(s) {
			t.Errorf("%q should be a valid stage", s)
		}
	}
	for _, s := range []string{"banana", "", "Applied", "interviewing"} {
		if isValidStage(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}

// The interaction shape now carries stage and notes alongside the timestamps.
func TestInteractionResponse_HasStageAndNotes(t *testing.T) {
	fields := marshalToFields(t, interactionResponse{JobID: 7})
	for _, want := range []string{"job_id", "viewed_at", "saved_at", "applied_at", "stage", "notes"} {
		if _, ok := fields[want]; !ok {
			t.Errorf("interactionResponse missing %q", want)
		}
	}
}
