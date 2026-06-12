package oauth

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestNewState_RandomAndURLSafe(t *testing.T) {
	a, err := NewState()
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	b, err := NewState()
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	if a == b {
		t.Error("two states are equal; want random")
	}
	if len(a) < 32 {
		t.Errorf("state %q too short", a)
	}
	for _, r := range a {
		if r == '+' || r == '/' || r == '=' {
			t.Errorf("state %q is not URL-safe", a)
		}
	}
}

func TestStateCookie_SetAndClear(t *testing.T) {
	app := fiber.New()
	app.Get("/set", func(c *fiber.Ctx) error {
		SetStateCookie(c, "abc", false)
		return nil
	})
	app.Get("/clear", func(c *fiber.Ctx) error {
		ClearStateCookie(c, false)
		return nil
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/set", nil))
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	set := resp.Header.Get("Set-Cookie")
	for _, want := range []string{StateCookieName + "=abc", "HttpOnly", "SameSite=Lax"} {
		if !strings.Contains(set, want) {
			t.Errorf("set cookie %q missing %q", set, want)
		}
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/clear", nil))
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	cleared := resp.Header.Get("Set-Cookie")
	if !strings.Contains(cleared, StateCookieName+"=") {
		t.Errorf("clear cookie %q does not clear %s", cleared, StateCookieName)
	}
}
