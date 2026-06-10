package config

import (
	"os"
	"time"
)

// Settings holds application configuration read from environment variables.
type Settings struct {
	Port           string
	DatabaseURL    string
	FrontendOrigin string

	// JWT settings for the auth surface. JWTSecret has no default: it is read
	// as-is and the server fails fast when it is empty (see cmd/server). The
	// enrich worker shares Load but ignores these, so the requirement lives at
	// the server entry point, not here.
	JWTSecret string
	JWTTTL    time.Duration
}

// Load reads configuration from the environment, falling back to sensible defaults.
func Load() Settings {
	return Settings{
		Port:           env("PORT", "8080"),
		DatabaseURL:    env("DATABASE_URL", "postgres://hire:hire@localhost:5432/hire?sslmode=disable"),
		FrontendOrigin: env("FRONTEND_ORIGIN", "http://localhost:5173"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTTTL:         envDuration("JWT_TTL", 24*time.Hour),
	}
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	// Reuse env()'s "unset or empty -> fallback" rule; an unparseable value
	// also falls back.
	if d, err := time.ParseDuration(env(key, "")); err == nil {
		return d
	}
	return fallback
}
