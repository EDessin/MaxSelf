package rest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/facade/application"
	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

func newFacadeHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	secret := "secret"
	token, err := httpx.IssueToken(secret, "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/register", "/auth/login":
			httpx.JSON(w, http.StatusOK, application.AuthResult{Token: token, User: application.User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"}})
		case "/users/me":
			if r.Header.Get("Authorization") != "Bearer "+token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			httpx.JSON(w, http.StatusOK, application.User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(identityServer.Close)

	activityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/activity-types":
			httpx.JSON(w, http.StatusOK, []application.ActivityRule{{Type: "exercise", Title: "Strength Session", XP: 40, Stat: "strength"}})
		case r.URL.Path == "/activities" && r.Method == http.MethodGet:
			httpx.JSON(w, http.StatusOK, []application.Activity{{ID: "activity-1", UserID: "user-1", Type: "exercise", Title: "Strength Session", XP: 40, Stat: "strength", OccurredAt: time.Now()}})
		case r.URL.Path == "/activities" && r.Method == http.MethodPost:
			httpx.JSON(w, http.StatusCreated, application.Activity{ID: "activity-2", UserID: "user-1", Type: "exercise", Title: "Strength Session", XP: 40, Stat: "strength", OccurredAt: time.Now()})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(activityServer.Close)

	progressServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/progress/user-1":
			httpx.JSON(w, http.StatusOK, application.Progress{UserID: "user-1", Level: 1, NextLevelXP: 100, Stats: map[string]int{}})
		case "/progress/award":
			httpx.JSON(w, http.StatusOK, application.Progress{UserID: "user-1", Level: 1, TotalXP: 40, NextLevelXP: 100, Stats: map[string]int{"strength": 40}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(progressServer.Close)

	service := application.NewService(
		application.NewClient(identityServer.URL, time.Second),
		application.NewClient(activityServer.URL, time.Second),
		application.NewClient(progressServer.URL, time.Second),
		secret,
	)
	handler := NewHandler(service, config.Config{IdentityServiceURL: identityServer.URL}).Routes()
	return handler, token
}

func TestFacadeRoutes(t *testing.T) {
	handler, token := newFacadeHandler(t)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "api") {
		t.Fatalf("unexpected health response: %d %s", recorder.Code, recorder.Body.String())
	}

	for _, route := range []string{"/api/auth/register", "/api/auth/login"} {
		recorder = httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, route, strings.NewReader(`{"email":"demo@example.com"}`)))
		if recorder.Code < 200 || recorder.Code >= 300 || !strings.Contains(recorder.Body.String(), `"token"`) {
			t.Fatalf("unexpected auth response for %s: %d %s", route, recorder.Code, recorder.Body.String())
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "demo@example.com") {
		t.Fatalf("unexpected me response: %d %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"rules"`) {
		t.Fatalf("unexpected dashboard response: %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/activity-types", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "exercise") {
		t.Fatalf("unexpected activity types response: %d %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/activities", strings.NewReader(`{"type":"exercise"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusGone || !strings.Contains(recorder.Body.String(), "manual XP claims are disabled") {
		t.Fatalf("unexpected create activity response: %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/auth/google/login", nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("expected google redirect, got %d", recorder.Code)
	}
}

func TestFacadeRoutesReturnErrors(t *testing.T) {
	handler, token := newFacadeHandler(t)

	for _, route := range []string{"/api/auth/register", "/api/auth/login"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, route, strings.NewReader(`{`)))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request for %s, got %d", route, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized me, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/dashboard", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized dashboard, got %d", recorder.Code)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/activities", strings.NewReader(`{"type":"exercise"}`))
	request.Header.Set("Authorization", "Bearer bad-token")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusGone {
		t.Fatalf("expected bad create activity request, got %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/integrations/google-health/connect", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected unconfigured connect, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/integrations/google-health/callback?state=bad&code=bad", nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("expected callback error redirect, got %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/integrations/google-health/sync", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected unconfigured sync, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/biometrics/waist-to-height", strings.NewReader(`{`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid waist request, got %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/biometrics/waist-to-height", strings.NewReader(`{"waistCentimeters":80,"heightCentimeters":180}`))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected unconfigured waist route, got %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/quest-claims/claim-1/claim", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected unconfigured quest claim route, got %d", recorder.Code)
	}
}

func TestIntegrationErrorStatuses(t *testing.T) {
	handler := NewHandler(application.Service{}, config.Config{})
	tests := []struct {
		err  error
		code int
	}{
		{application.ErrGoogleHealthNotConfigured, http.StatusNotImplemented},
		{application.ErrGoogleHealthNotConnected, http.StatusConflict},
		{application.ErrQuestClaimNotFound, http.StatusNotFound},
		{application.ErrQuestClaimAlreadyClaimed, http.StatusConflict},
		{errors.New("other"), http.StatusBadRequest},
	}
	for _, tt := range tests {
		recorder := httptest.NewRecorder()
		handler.integrationError(recorder, tt.err)
		if recorder.Code != tt.code {
			t.Fatalf("expected %d for %v, got %d", tt.code, tt.err, recorder.Code)
		}
	}
}
