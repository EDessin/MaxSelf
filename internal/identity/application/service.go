package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EDessin/MaxSelf/internal/identity/domain"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailRequired      = errors.New("email is required")
	ErrPasswordRequired   = errors.New("password is required")
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User, identity domain.AuthIdentity) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindByProvider(ctx context.Context, provider domain.Provider, providerUserID string) (domain.User, error)
	LinkIdentity(ctx context.Context, identity domain.AuthIdentity) error
}

type Service struct {
	users          UserRepository
	jwtSecret      string
	tokenExpiresIn time.Duration
}

type AuthResult struct {
	Token string     `json:"token"`
	User  PublicUser `json:"user"`
}

type PublicUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
}

func NewService(users UserRepository, jwtSecret string, tokenExpiresIn time.Duration) Service {
	return Service{users: users, jwtSecret: jwtSecret, tokenExpiresIn: tokenExpiresIn}
}

func (s Service) Register(ctx context.Context, email, password, displayName string) (AuthResult, error) {
	email = domain.NormalizeEmail(email)
	if email == "" {
		return AuthResult{}, ErrEmailRequired
	}
	if password == "" {
		return AuthResult{}, ErrPasswordRequired
	}
	if displayName == "" {
		displayName = email
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	userID := uuid.NewString()
	user := domain.User{
		ID:           userID,
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: string(hash),
	}
	identity := domain.AuthIdentity{
		ID:             uuid.NewString(),
		UserID:         userID,
		Provider:       domain.ProviderEmail,
		ProviderUserID: email,
		Email:          email,
	}
	created, err := s.users.CreateUser(ctx, user, identity)
	if err != nil {
		return AuthResult{}, err
	}
	return s.authResult(created)
}

func (s Service) Login(ctx context.Context, email, password string) (AuthResult, error) {
	user, err := s.users.FindByEmail(ctx, domain.NormalizeEmail(email))
	if err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	return s.authResult(user)
}

func (s Service) LoginWithProvider(ctx context.Context, provider domain.Provider, providerUserID, email, displayName, avatarURL string) (AuthResult, error) {
	user, err := s.users.FindByProvider(ctx, provider, providerUserID)
	if err == nil {
		return s.authResult(user)
	}

	email = domain.NormalizeEmail(email)
	user, err = s.users.FindByEmail(ctx, email)
	if err == nil {
		identity := domain.AuthIdentity{
			ID:             uuid.NewString(),
			UserID:         user.ID,
			Provider:       provider,
			ProviderUserID: providerUserID,
			Email:          email,
		}
		if linkErr := s.users.LinkIdentity(ctx, identity); linkErr != nil {
			return AuthResult{}, linkErr
		}
		return s.authResult(user)
	}

	if displayName == "" {
		displayName = email
	}
	userID := uuid.NewString()
	newUser := domain.User{
		ID:          userID,
		Email:       email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
	identity := domain.AuthIdentity{
		ID:             uuid.NewString(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
	}
	created, err := s.users.CreateUser(ctx, newUser, identity)
	if err != nil {
		return AuthResult{}, err
	}
	return s.authResult(created)
}

func (s Service) Me(ctx context.Context, claims httpx.Claims) (PublicUser, error) {
	user, err := s.users.FindByEmail(ctx, claims.Email)
	if err != nil {
		return PublicUser{}, err
	}
	return publicUser(user), nil
}

func (s Service) authResult(user domain.User) (AuthResult, error) {
	token, err := httpx.IssueToken(s.jwtSecret, user.ID, user.Email, user.DisplayName, s.tokenExpiresIn)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Token: token, User: publicUser(user)}, nil
}

func publicUser(user domain.User) PublicUser {
	return PublicUser{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
	}
}
