package handler

import (
	"context"
	"errors"
	"log"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/auth/oauth"
	"github.com/strelov1/freehire/internal/db"
)

// ListOAuthProviders returns the names of enabled OAuth providers, so the SPA
// renders only usable sign-in buttons.
func (h *Handler) ListOAuthProviders(c *fiber.Ctx) error {
	names := make([]string, 0, len(h.oauth))
	for name := range h.oauth {
		names = append(names, name)
	}
	sort.Strings(names)
	return c.JSON(fiber.Map{"data": names})
}

// OAuthStart begins the authorization-code flow: it stores a fresh CSRF state
// in a short-lived cookie and redirects the browser to the provider's consent
// page carrying the same state.
func (h *Handler) OAuthStart(c *fiber.Ctx) error {
	p, ok := h.oauth[c.Params("provider")]
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "unknown provider")
	}

	state, err := oauth.NewState()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start sign-in")
	}
	oauth.SetStateCookie(c, state, h.cookieSecure)
	return c.Redirect(p.AuthCodeURL(state), fiber.StatusFound)
}

// OAuthCallback completes the flow: verify the CSRF state, exchange the code
// for the provider identity, resolve (or create) the account, start the
// session, and send the browser back to the SPA. The callback is a top-level
// navigation, so every failure redirects with auth_error instead of rendering
// JSON; details go to the server log.
func (h *Handler) OAuthCallback(c *fiber.Ctx) error {
	p, ok := h.oauth[c.Params("provider")]
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "unknown provider")
	}

	// The state is single-use: clear the cookie no matter how the rest goes.
	cookieState := c.Cookies(oauth.StateCookieName)
	oauth.ClearStateCookie(c, h.cookieSecure)

	state, code := c.Query("state"), c.Query("code")
	if state == "" || state != cookieState {
		return h.oauthFail(c, p.Name(), errors.New("state mismatch"))
	}
	if code == "" {
		return h.oauthFail(c, p.Name(), errors.New("missing code"))
	}

	identity, err := p.FetchIdentity(c.Context(), code)
	if err != nil {
		return h.oauthFail(c, p.Name(), err)
	}

	userID, err := h.resolveOAuthUser(c.Context(), p.Name(), identity)
	if err != nil {
		return h.oauthFail(c, p.Name(), err)
	}

	if err := h.setSession(c, userID); err != nil {
		return h.oauthFail(c, p.Name(), err)
	}
	return c.Redirect(h.frontendOrigin+"/", fiber.StatusFound)
}

// oauthFail logs the failure server-side and sends the browser to the SPA
// with the generic auth_error marker (never a JSON error page).
func (h *Handler) oauthFail(c *fiber.Ctx, provider string, err error) error {
	log.Printf("oauth %s: sign-in failed: %v", provider, err)
	return c.Redirect(h.frontendOrigin+"/?auth_error=oauth", fiber.StatusFound)
}

// resolveOAuthUser turns a provider identity into a local user id:
//  1. an existing identity resolves directly (returning user);
//  2. otherwise a verified email is required — it links to the existing
//     account with that email, or creates a new passwordless account —
//     identity insert and any user insert run in one transaction.
//
// A unique-violation race (concurrent duplicate callback) retries the
// identity lookup once; the DB constraints are the backstop.
func (h *Handler) resolveOAuthUser(ctx context.Context, provider string, identity oauth.Identity) (int64, error) {
	lookup := db.GetUserByIdentityParams{Provider: provider, ProviderUserID: identity.ProviderUserID}

	user, err := h.queries.GetUserByIdentity(ctx, lookup)
	if err == nil {
		return user.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	// First sign-in with this identity: only a verified email may key or link
	// an account (an unverified one would allow account takeover).
	if !identity.EmailVerified || identity.Email == "" {
		return 0, errors.New("no verified email")
	}
	email := strings.ToLower(strings.TrimSpace(identity.Email))

	userID, err := h.linkOrCreateUser(ctx, lookup, email)
	if isUniqueViolation(err) {
		// Lost a race with a concurrent callback for the same identity.
		if user, retryErr := h.queries.GetUserByIdentity(ctx, lookup); retryErr == nil {
			return user.ID, nil
		}
		return 0, err
	}
	return userID, err
}

// linkOrCreateUser links the identity to the account matching email, creating
// a passwordless account when none exists, in a single transaction.
func (h *Handler) linkOrCreateUser(ctx context.Context, identity db.GetUserByIdentityParams, email string) (int64, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := h.queries.WithTx(tx)

	var userID int64
	existing, err := q.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		userID = existing.ID
	case errors.Is(err, pgx.ErrNoRows):
		created, createErr := q.CreateUser(ctx, db.CreateUserParams{
			Email:        email,
			PasswordHash: pgtype.Text{}, // passwordless account
		})
		if createErr != nil {
			return 0, createErr
		}
		userID = created.ID
	default:
		return 0, err
	}

	if err := q.CreateUserIdentity(ctx, db.CreateUserIdentityParams{
		Provider:       identity.Provider,
		ProviderUserID: identity.ProviderUserID,
		UserID:         userID,
	}); err != nil {
		return 0, err
	}
	return userID, tx.Commit(ctx)
}
