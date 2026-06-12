// Package oauth implements sign-in via external OAuth providers (Google,
// GitHub, LinkedIn) behind a small Provider interface and a registry built
// from config, mirroring how internal/sources grows by adapters. Providers
// are read-only identity fetchers: tokens are used once to resolve who the
// user is and are never stored.
package oauth

import (
	"context"

	"github.com/strelov1/freehire/internal/config"
)

// Identity is what a provider knows about the signing-in user: a stable
// per-provider user id and, when available, an email with its verification
// status. Account resolution MUST ignore unverified emails (linking on an
// unverified email would allow account takeover).
type Identity struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
}

// Provider is one OAuth provider in the authorization-code flow: it builds
// the consent URL and turns a callback code into an Identity.
type Provider interface {
	Name() string
	AuthCodeURL(state string) string
	FetchIdentity(ctx context.Context, code string) (Identity, error)
}

// NewRegistry builds the enabled-provider map from per-provider credentials.
// A provider is enabled only when both client id and secret are set; unknown
// provider names are ignored. Redirect URLs derive from origin (the
// same-origin SPA/API base), so each provider's registered callback is
// origin + /api/v1/auth/oauth/<name>/callback.
func NewRegistry(origin string, creds map[string]config.OAuthCredentials) map[string]Provider {
	constructors := map[string]func(clientID, clientSecret, redirectURL string) Provider{
		"google":   NewGoogle,
		"github":   NewGitHub,
		"linkedin": NewLinkedIn,
	}

	reg := make(map[string]Provider)
	for name, c := range creds {
		build, known := constructors[name]
		if !known || c.ClientID == "" || c.ClientSecret == "" {
			continue
		}
		reg[name] = build(c.ClientID, c.ClientSecret, origin+"/api/v1/auth/oauth/"+name+"/callback")
	}
	return reg
}
