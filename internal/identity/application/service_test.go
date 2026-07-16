package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/identity/domain"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepository struct {
	usersByEmail    map[string]domain.User
	usersByProvider map[string]domain.User
	createErr       error
	linkErr         error
	createdUser     domain.User
	createdIdentity domain.AuthIdentity
	linkedIdentity  domain.AuthIdentity
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		usersByEmail:    map[string]domain.User{},
		usersByProvider: map[string]domain.User{},
	}
}

func (r *fakeUserRepository) CreateUser(_ context.Context, user domain.User, identity domain.AuthIdentity) (domain.User, error) {
	r.createdUser = user
	r.createdIdentity = identity
	if r.createErr != nil {
		return domain.User{}, r.createErr
	}
	r.usersByEmail[user.Email] = user
	r.usersByProvider[string(identity.Provider)+":"+identity.ProviderUserID] = user
	return user, nil
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (domain.User, error) {
	user, ok := r.usersByEmail[email]
	if !ok {
		return domain.User{}, errors.New("not found")
	}
	return user, nil
}

func (r *fakeUserRepository) FindByProvider(_ context.Context, provider domain.Provider, providerUserID string) (domain.User, error) {
	user, ok := r.usersByProvider[string(provider)+":"+providerUserID]
	if !ok {
		return domain.User{}, errors.New("not found")
	}
	return user, nil
}

func (r *fakeUserRepository) LinkIdentity(_ context.Context, identity domain.AuthIdentity) error {
	r.linkedIdentity = identity
	return r.linkErr
}

func TestRegisterValidatesAndCreatesUser(t *testing.T) {
	service := NewService(newFakeUserRepository(), "secret", time.Hour)

	if _, err := service.Register(context.Background(), " ", "password", ""); !errors.Is(err, ErrEmailRequired) {
		t.Fatalf("expected email required, got %v", err)
	}
	if _, err := service.Register(context.Background(), "demo@example.com", "", ""); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected password required, got %v", err)
	}

	repo := newFakeUserRepository()
	service = NewService(repo, "secret", time.Hour)
	result, err := service.Register(context.Background(), " DEMO@Example.COM ", "maxself", "")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if result.Token == "" || result.User.Email != "demo@example.com" || result.User.DisplayName != "demo@example.com" {
		t.Fatalf("unexpected auth result: %+v", result)
	}
	if repo.createdUser.ID == "" || repo.createdIdentity.Provider != domain.ProviderEmail {
		t.Fatalf("user/identity not created: %+v %+v", repo.createdUser, repo.createdIdentity)
	}
	if bcrypt.CompareHashAndPassword([]byte(repo.createdUser.PasswordHash), []byte("maxself")) != nil {
		t.Fatal("password hash does not match original password")
	}
}

func TestLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("maxself"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo := newFakeUserRepository()
	repo.usersByEmail["demo@example.com"] = domain.User{
		ID:           "user-1",
		Email:        "demo@example.com",
		DisplayName:  "Demo",
		PasswordHash: string(hash),
	}
	service := NewService(repo, "secret", time.Hour)

	result, err := service.Login(context.Background(), " Demo@Example.com ", "maxself")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.Token == "" || result.User.ID != "user-1" {
		t.Fatalf("unexpected login result: %+v", result)
	}

	if _, err := service.Login(context.Background(), "missing@example.com", "maxself"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials for missing user, got %v", err)
	}
	if _, err := service.Login(context.Background(), "demo@example.com", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials for wrong password, got %v", err)
	}
}

func TestLoginWithProviderUsesExistingLinksAndCreatesWhenNeeded(t *testing.T) {
	repo := newFakeUserRepository()
	existing := domain.User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"}
	repo.usersByProvider["google:google-1"] = existing
	service := NewService(repo, "secret", time.Hour)

	result, err := service.LoginWithProvider(context.Background(), domain.ProviderGoogle, "google-1", "demo@example.com", "", "")
	if err != nil {
		t.Fatalf("LoginWithProvider existing link returned error: %v", err)
	}
	if result.User.ID != "user-1" {
		t.Fatalf("expected linked user, got %+v", result.User)
	}

	repo = newFakeUserRepository()
	repo.usersByEmail["demo@example.com"] = existing
	service = NewService(repo, "secret", time.Hour)
	result, err = service.LoginWithProvider(context.Background(), domain.ProviderGoogle, "google-2", " Demo@Example.com ", "", "")
	if err != nil {
		t.Fatalf("LoginWithProvider existing email returned error: %v", err)
	}
	if result.User.ID != "user-1" || repo.linkedIdentity.UserID != "user-1" || repo.linkedIdentity.ProviderUserID != "google-2" {
		t.Fatalf("expected provider identity link, got result=%+v link=%+v", result, repo.linkedIdentity)
	}

	repo = newFakeUserRepository()
	service = NewService(repo, "secret", time.Hour)
	result, err = service.LoginWithProvider(context.Background(), domain.ProviderGoogle, "google-3", " NEW@Example.com ", "", "avatar.png")
	if err != nil {
		t.Fatalf("LoginWithProvider new user returned error: %v", err)
	}
	if result.User.Email != "new@example.com" || result.User.DisplayName != "new@example.com" || result.User.AvatarURL != "avatar.png" {
		t.Fatalf("unexpected new provider user: %+v", result.User)
	}
}

func TestLoginWithProviderPropagatesRepositoryErrors(t *testing.T) {
	repo := newFakeUserRepository()
	repo.usersByEmail["demo@example.com"] = domain.User{ID: "user-1", Email: "demo@example.com"}
	repo.linkErr = errors.New("link failed")
	service := NewService(repo, "secret", time.Hour)

	if _, err := service.LoginWithProvider(context.Background(), domain.ProviderGoogle, "google-1", "demo@example.com", "", ""); !errors.Is(err, repo.linkErr) {
		t.Fatalf("expected link error, got %v", err)
	}

	repo = newFakeUserRepository()
	repo.createErr = errors.New("create failed")
	service = NewService(repo, "secret", time.Hour)
	if _, err := service.LoginWithProvider(context.Background(), domain.ProviderGoogle, "google-1", "demo@example.com", "Demo", ""); !errors.Is(err, repo.createErr) {
		t.Fatalf("expected create error, got %v", err)
	}
}

func TestMeLooksUpPublicUserFromClaims(t *testing.T) {
	repo := newFakeUserRepository()
	repo.usersByEmail["demo@example.com"] = domain.User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo", AvatarURL: "avatar.png"}
	service := NewService(repo, "secret", time.Hour)

	user, err := service.Me(context.Background(), httpx.Claims{Email: "demo@example.com"})
	if err != nil {
		t.Fatalf("Me returned error: %v", err)
	}
	if user.ID != "user-1" || user.AvatarURL != "avatar.png" {
		t.Fatalf("unexpected public user: %+v", user)
	}
}
