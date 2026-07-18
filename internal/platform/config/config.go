package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName             string
	Port                    string
	DatabaseURL             string
	JWTSecret               string
	FrontendURL             string
	IdentityServiceURL      string
	ActivityServiceURL      string
	ProgressServiceURL      string
	GoogleClientID          string
	GoogleClientSecret      string
	GoogleRedirectURL       string
	GoogleHealthRedirectURL string
	GoogleHealthAPIBaseURL  string
	GoogleHealthTimeout     time.Duration
	HTTPClientTimeout       time.Duration
	AccessTokenExpiresIn    time.Duration
}

func Load(serviceName, defaultPort string) Config {
	return Config{
		ServiceName:             serviceName,
		Port:                    env("PORT", defaultPort),
		DatabaseURL:             env("DATABASE_URL", "postgres://maxself:maxself@localhost:5432/maxself?sslmode=disable"),
		JWTSecret:               env("JWT_SECRET", "dev-only-change-me"),
		FrontendURL:             env("FRONTEND_URL", "http://localhost:4201"),
		IdentityServiceURL:      env("IDENTITY_SERVICE_URL", "http://localhost:8081"),
		ActivityServiceURL:      env("ACTIVITY_SERVICE_URL", "http://localhost:8082"),
		ProgressServiceURL:      env("PROGRESS_SERVICE_URL", "http://localhost:8083"),
		GoogleClientID:          env("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:      env("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:       env("GOOGLE_REDIRECT_URL", "http://localhost:8081/auth/google/callback"),
		GoogleHealthRedirectURL: env("GOOGLE_HEALTH_REDIRECT_URL", "http://localhost:8080/api/integrations/google-health/callback"),
		GoogleHealthAPIBaseURL:  env("GOOGLE_HEALTH_API_BASE_URL", "https://health.googleapis.com"),
		GoogleHealthTimeout:     secondsEnv("GOOGLE_HEALTH_HTTP_CLIENT_TIMEOUT_SECONDS", 30),
		HTTPClientTimeout:       secondsEnv("HTTP_CLIENT_TIMEOUT_SECONDS", 5),
		AccessTokenExpiresIn:    time.Duration(secondsEnv("ACCESS_TOKEN_EXPIRES_SECONDS", 86400).Seconds()) * time.Second,
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func secondsEnv(key string, fallback int) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return time.Duration(fallback) * time.Second
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(value) * time.Second
}
