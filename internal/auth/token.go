package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer mints and verifies stateless HS256 JWTs that carry the user id as the
// subject. One Issuer is built from the configured secret and TTL and shared
// across requests.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

// NewIssuer returns an Issuer signing with secret and stamping each token to
// expire after ttl.
func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

// Issue returns a signed token whose subject is userID, expiring after the
// Issuer's TTL.
func (i *Issuer) Issue(userID int64) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
}

// Parse verifies a token's signature and expiry and returns its subject user
// id. It rejects any token not signed with HMAC (guarding against algorithm
// confusion) and any whose subject is not a valid id.
func (i *Issuer) Parse(token string) (int64, error) {
	claims := &jwt.RegisteredClaims{}
	if _, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return i.secret, nil
	}); err != nil {
		return 0, err
	}

	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid token subject: %w", err)
	}
	return id, nil
}
