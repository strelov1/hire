package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// localsUserID is the c.Locals key under which RequireAuth stores the
// authenticated user id. Handlers read it via UserID.
const localsUserID = "auth.userID"

// RequireAuth returns middleware that validates the `Authorization: Bearer`
// token and stores the resolved user id in the request locals. It responds 401
// on a missing, malformed, expired, or invalid token.
func RequireAuth(iss *Issuer) fiber.Handler {
	const prefix = "Bearer "
	return func(c *fiber.Ctx) error {
		h := c.Get(fiber.HeaderAuthorization)
		if !strings.HasPrefix(h, prefix) {
			return fiber.NewError(fiber.StatusUnauthorized, "missing or malformed authorization header")
		}
		id, err := iss.Parse(strings.TrimPrefix(h, prefix))
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}
		c.Locals(localsUserID, id)
		return c.Next()
	}
}

// UserID returns the authenticated user id stored by RequireAuth. The second
// result is false when the request did not pass through RequireAuth.
func UserID(c *fiber.Ctx) (int64, bool) {
	id, ok := c.Locals(localsUserID).(int64)
	return id, ok
}
