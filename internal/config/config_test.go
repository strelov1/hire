package config

import (
	"testing"
	"time"
)

func TestLoad_JWTSecretFromEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "s3cret")

	if got := Load().JWTSecret; got != "s3cret" {
		t.Errorf("JWTSecret = %q, want %q", got, "s3cret")
	}
}

func TestLoad_JWTTTLDefaultsWhenUnset(t *testing.T) {
	t.Setenv("JWT_TTL", "")

	if got := Load().JWTTTL; got != 24*time.Hour {
		t.Errorf("JWTTTL = %v, want 24h", got)
	}
}

func TestLoad_JWTTTLParsesDuration(t *testing.T) {
	t.Setenv("JWT_TTL", "1h30m")

	if got := Load().JWTTTL; got != 90*time.Minute {
		t.Errorf("JWTTTL = %v, want 1h30m", got)
	}
}

func TestLoad_JWTTTLFallsBackOnGarbage(t *testing.T) {
	t.Setenv("JWT_TTL", "not-a-duration")

	if got := Load().JWTTTL; got != 24*time.Hour {
		t.Errorf("JWTTTL = %v, want 24h fallback", got)
	}
}
