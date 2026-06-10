package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/hire/internal/auth"
)

// registerApp mounts only the register route on a handler with no DB. The
// validation cases below all reject before any query runs, so a nil queries is
// never dereferenced.
func registerApp() *fiber.App {
	app := fiber.New()
	h := &Handler{issuer: auth.NewIssuer("test-secret", time.Hour)}
	app.Post("/register", h.Register)
	return app
}

func postJSON(t *testing.T, app *fiber.App, path, body string) int {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodPost, path, strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	return resp.StatusCode
}

func TestRegister_RejectsShortPassword(t *testing.T) {
	if got := postJSON(t, registerApp(), "/register", `{"email":"a@b.com","password":"short"}`); got != fiber.StatusBadRequest {
		t.Errorf("status = %d, want 400", got)
	}
}

func TestRegister_RejectsInvalidEmail(t *testing.T) {
	if got := postJSON(t, registerApp(), "/register", `{"email":"not-an-email","password":"longenough123"}`); got != fiber.StatusBadRequest {
		t.Errorf("status = %d, want 400", got)
	}
}

func TestRegister_RejectsMalformedBody(t *testing.T) {
	if got := postJSON(t, registerApp(), "/register", `{not json`); got != fiber.StatusBadRequest {
		t.Errorf("status = %d, want 400", got)
	}
}

// userResponse is the only user shape that reaches a response. This locks the
// contract that it never carries the password hash.
func TestUserResponse_OmitsPasswordHash(t *testing.T) {
	raw, err := json.Marshal(userResponse{ID: 1, Email: "a@b.com"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, leaked := fields["password_hash"]; leaked {
		t.Error("userResponse must not include password_hash")
	}
	for _, want := range []string{"id", "email", "created_at"} {
		if _, ok := fields[want]; !ok {
			t.Errorf("userResponse missing %q", want)
		}
	}
}
