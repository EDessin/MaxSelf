package domain

import (
	"strings"
	"time"
)

type Provider string

const (
	ProviderEmail  Provider = "email"
	ProviderGoogle Provider = "google"
)

type User struct {
	ID           string
	Email        string
	DisplayName  string
	AvatarURL    string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AuthIdentity struct {
	ID             string
	UserID         string
	Provider       Provider
	ProviderUserID string
	Email          string
	CreatedAt      time.Time
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
