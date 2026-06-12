package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v2"
)

// StateCookieName carries the CSRF state between the start redirect and the
// provider callback. Lax is enough: the callback is a top-level GET
// navigation, on which Lax cookies are sent.
const StateCookieName = "hire_oauth_state"

// stateTTL bounds how long a started sign-in stays completable. Ten minutes
// covers a slow consent screen without leaving stale states around.
const stateTTL = 10 * time.Minute

// NewState returns a fresh URL-safe random state value.
func NewState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SetStateCookie stores the state for the upcoming callback to verify.
func SetStateCookie(c *fiber.Ctx, state string, secure bool) {
	writeStateCookie(c, state, time.Now().Add(stateTTL), secure)
}

// ClearStateCookie removes the state cookie (the state is single-use).
func ClearStateCookie(c *fiber.Ctx, secure bool) {
	writeStateCookie(c, "", time.Now().Add(-time.Hour), secure)
}

// writeStateCookie is the single place the cookie's attributes are set, so
// set and clear can't drift apart (same pattern as the session cookie).
func writeStateCookie(c *fiber.Ctx, value string, expires time.Time, secure bool) {
	c.Cookie(&fiber.Cookie{
		Name:     StateCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}
