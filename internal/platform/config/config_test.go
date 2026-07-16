package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("FRONTEND_URL", "")
	t.Setenv("IDENTITY_SERVICE_URL", "")
	t.Setenv("ACTIVITY_SERVICE_URL", "")
	t.Setenv("PROGRESS_SERVICE_URL", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_REDIRECT_URL", "")
	t.Setenv("HTTP_CLIENT_TIMEOUT_SECONDS", "")
	t.Setenv("ACCESS_TOKEN_EXPIRES_SECONDS", "")

	cfg := Load("api", "8080")
	if cfg.ServiceName != "api" || cfg.Port != "8080" || cfg.JWTSecret == "" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.HTTPClientTimeout != 5*time.Second || cfg.AccessTokenExpiresIn != 24*time.Hour {
		t.Fatalf("unexpected default durations: %+v", cfg)
	}
}

func TestLoadOverridesAndInvalidDurations(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("HTTP_CLIENT_TIMEOUT_SECONDS", "12")
	t.Setenv("ACCESS_TOKEN_EXPIRES_SECONDS", "bad")

	cfg := Load("api", "8080")
	if cfg.Port != "9000" || cfg.JWTSecret != "secret" || cfg.HTTPClientTimeout != 12*time.Second {
		t.Fatalf("unexpected overrides: %+v", cfg)
	}
	if cfg.AccessTokenExpiresIn != 24*time.Hour {
		t.Fatalf("invalid duration should use fallback, got %s", cfg.AccessTokenExpiresIn)
	}
}
