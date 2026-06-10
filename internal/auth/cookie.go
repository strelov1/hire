package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// CookieName is the httpOnly cookie that carries the JWT. The SPA never reads
// it (it can't — HTTPOnly); the browser attaches it automatically on same-site
// requests, and RequireAuth reads it server-side.
const CookieName = "hire_token"

// SetTokenCookie writes the auth cookie. SameSite=Lax (with a same-origin
// deployment) sends it on the app's own requests while blocking it on
// cross-site ones, which covers CSRF for the current endpoints. secure comes
// from config so dev (http://localhost) and HTTPS deployments both work.
func SetTokenCookie(c *fiber.Ctx, token string, ttl time.Duration, secure bool) {
	writeTokenCookie(c, token, time.Now().Add(ttl), secure)
}

// ClearTokenCookie expires the auth cookie (logout).
func ClearTokenCookie(c *fiber.Ctx, secure bool) {
	writeTokenCookie(c, "", time.Now().Add(-time.Hour), secure)
}

// writeTokenCookie is the single place the cookie's attributes are set, so set
// and clear can't drift apart (the browser only overwrites a cookie whose
// attributes match).
func writeTokenCookie(c *fiber.Ctx, value string, expires time.Time, secure bool) {
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}
