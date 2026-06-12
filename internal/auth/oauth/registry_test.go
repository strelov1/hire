package oauth

import (
	"strings"
	"testing"

	"github.com/strelov1/freehire/internal/config"
)

func TestNewRegistry_OnlyCompleteCredentialsEnable(t *testing.T) {
	reg := NewRegistry("http://localhost:5173", map[string]config.OAuthCredentials{
		"google":   {ClientID: "id", ClientSecret: "secret"},
		"github":   {ClientID: "id"}, // missing secret -> disabled
		"linkedin": {},               // unset -> disabled
	})

	if _, ok := reg["google"]; !ok {
		t.Error("google missing; want enabled")
	}
	if _, ok := reg["github"]; ok {
		t.Error("github enabled; want disabled (no secret)")
	}
	if _, ok := reg["linkedin"]; ok {
		t.Error("linkedin enabled; want disabled (unset)")
	}
}

func TestNewRegistry_IgnoresUnknownProvider(t *testing.T) {
	reg := NewRegistry("http://x", map[string]config.OAuthCredentials{
		"myspace": {ClientID: "id", ClientSecret: "secret"},
	})
	if len(reg) != 0 {
		t.Errorf("registry = %v, want empty", reg)
	}
}

func TestNewRegistry_RedirectURLDerivesFromOrigin(t *testing.T) {
	reg := NewRegistry("https://freehire.dev", map[string]config.OAuthCredentials{
		"google": {ClientID: "id", ClientSecret: "secret"},
	})
	u := reg["google"].AuthCodeURL("s")
	if !strings.Contains(u, "freehire.dev%2Fapi%2Fv1%2Fauth%2Foauth%2Fgoogle%2Fcallback") {
		t.Errorf("AuthCodeURL %q does not carry the derived redirect URL", u)
	}
}
