package rest

import (
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
			httpx.JSON(w, http.StatusOK, []application.ActivityRule{{Type: "exercise", Title: "Move Your Body", XP: 40, Stat: "strength"}})
		case r.URL.Path == "/activities" && r.Method == http.MethodGet:
			httpx.JSON(w, http.StatusOK, []application.Activity{{ID: "activity-1", UserID: "user-1", Type: "exercise", Title: "Move Your Body", XP: 40, Stat: "strength", OccurredAt: time.Now()}})
		case r.URL.Path == "/activities" && r.Method == http.MethodPost:
			httpx.JSON(w, http.StatusCreated, application.Activity{ID: "activity-2", UserID: "user-1", Type: "exercise", Title: "Move Your Body", XP: 40, Stat: "strength", OccurredAt: time.Now()})
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
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), `"progress"`) {
		t.Fatalf("unexpected create activity response: %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/auth/google/login", nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("expected google redirect, got %d", recorder.Code)
	}
}

func TestFacadeRoutesReturnErrors(t *testing.T) {
	handler, _ := newFacadeHandler(t)

	for _, route := range []string{"/api/auth/register", "/api/auth/login", "/api/activities"} {
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
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected bad create activity request, got %d", recorder.Code)
	}
}
