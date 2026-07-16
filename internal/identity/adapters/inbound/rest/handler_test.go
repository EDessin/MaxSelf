package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/identity/application"
	"github.com/EDessin/MaxSelf/internal/identity/domain"
	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

type fakeUserRepository struct {
	users map[string]domain.User
}

func (r *fakeUserRepository) CreateUser(_ context.Context, user domain.User, _ domain.AuthIdentity) (domain.User, error) {
	r.users[user.Email] = user
	return user, nil
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (domain.User, error) {
	user, ok := r.users[email]
	if !ok {
		return domain.User{}, http.ErrNoCookie
	}
	return user, nil
}

func (r *fakeUserRepository) FindByProvider(_ context.Context, _ domain.Provider, _ string) (domain.User, error) {
	return domain.User{}, http.ErrNoCookie
}

func (r *fakeUserRepository) LinkIdentity(_ context.Context, _ domain.AuthIdentity) error {
	return nil
}

func TestIdentityRoutes(t *testing.T) {
	repo := &fakeUserRepository{users: map[string]domain.User{}}
	cfg := config.Config{JWTSecret: "secret", GoogleRedirectURL: "http://identity/callback", FrontendURL: "http://frontend"}
	service := application.NewService(repo, cfg.JWTSecret, time.Hour)
	handler := NewHandler(service, cfg).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "identity") {
		t.Fatalf("unexpected health response: %d %s", recorder.Code, recorder.Body.String())
	}

	registerBody := `{"email":"demo@example.com","password":"maxself","displayName":"Demo"}`
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(registerBody)))
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), `"token"`) {
		t.Fatalf("unexpected register response: %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"demo@example.com","password":"maxself"}`)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"token"`) {
		t.Fatalf("unexpected login response: %d %s", recorder.Code, recorder.Body.String())
	}

	token, err := httpx.IssueToken(cfg.JWTSecret, "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "demo@example.com") {
		t.Fatalf("unexpected me response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestIdentityRoutesReturnErrors(t *testing.T) {
	cfg := config.Config{JWTSecret: "secret", FrontendURL: "http://frontend"}
	handler := NewHandler(application.NewService(&fakeUserRepository{users: map[string]domain.User{}}, cfg.JWTSecret, time.Hour), cfg).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"demo@example.com","unknown":true}`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid register request, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"missing@example.com","password":"nope"}`)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected login unauthorized, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid login request, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/auth/google/login", nil))
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected google not configured, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil))
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected google callback not configured, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/me", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized me, got %d", recorder.Code)
	}

	token, err := httpx.IssueToken(cfg.JWTSecret, "user-1", "missing@example.com", "Missing", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing user not found, got %d", recorder.Code)
	}
}

func TestGoogleRoutesWhenConfigured(t *testing.T) {
	cfg := config.Config{
		JWTSecret:          "secret",
		FrontendURL:        "http://frontend",
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "http://identity/auth/google/callback",
	}
	handler := NewHandler(application.NewService(&fakeUserRepository{users: map[string]domain.User{}}, cfg.JWTSecret, time.Hour), cfg).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/auth/google/login", nil))
	if recorder.Code != http.StatusFound || !strings.Contains(recorder.Header().Get("Location"), "accounts.google.com") {
		t.Fatalf("expected Google auth redirect, got %d %s", recorder.Code, recorder.Header().Get("Location"))
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil))
	if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "http://frontend/login?error=google" {
		t.Fatalf("expected callback error redirect, got %d %s", recorder.Code, recorder.Header().Get("Location"))
	}
}

func TestGoogleCallbackExchangeFailures(t *testing.T) {
	cfg := config.Config{
		JWTSecret:          "secret",
		FrontendURL:        "http://frontend",
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "http://identity/auth/google/callback",
	}
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access-token","token_type":"Bearer"}`))
	}))
	defer tokenServer.Close()

	handler := NewHandler(application.NewService(&fakeUserRepository{users: map[string]domain.User{}}, cfg.JWTSecret, time.Hour), cfg)
	handler.oauth.Endpoint.TokenURL = tokenServer.URL
	recorder := httptest.NewRecorder()
	handler.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=abc", nil))
	if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "http://frontend/login?error=google" {
		t.Fatalf("expected missing id_token redirect, got %d %s", recorder.Code, recorder.Header().Get("Location"))
	}

	tokenServer.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access-token","token_type":"Bearer","id_token":"not-a-token"}`))
	})
	recorder = httptest.NewRecorder()
	handler.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=abc", nil))
	if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "http://frontend/login?error=google" {
		t.Fatalf("expected invalid id_token redirect, got %d %s", recorder.Code, recorder.Header().Get("Location"))
	}
}
