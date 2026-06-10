// Package auth holds the security primitives for the user-auth surface:
// password hashing, stateless JWT issue/verify, and the Fiber middleware that
// guards protected routes. Handlers depend on these through small functions so
// the crypto and token logic stays testable in isolation.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of plain. The salt and cost are embedded in
// the returned string, so callers store only this single value.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword reports whether plain matches a bcrypt hash. It returns nil on a
// match and a non-nil error otherwise (including a malformed hash).
func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
