package application

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

func TestServiceDashboardAndCreateActivity(t *testing.T) {
	secret := "secret"
	token, err := httpx.IssueToken(secret, "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/register" || r.URL.Path == "/auth/login" {
			httpx.JSON(w, http.StatusOK, AuthResult{Token: token, User: User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"}})
			return
		}
		if r.URL.Path == "/users/me" && r.Header.Get("Authorization") == "Bearer "+token {
			httpx.JSON(w, http.StatusOK, User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"})
			return
		}
		http.NotFound(w, r)
	}))
	defer identityServer.Close()

	activityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/activity-types":
			httpx.JSON(w, http.StatusOK, []ActivityRule{{Type: "exercise", Title: "Move Your Body", XP: 40, Stat: "strength"}})
		case r.URL.Path == "/activities" && r.Method == http.MethodGet:
			if r.Header.Get("X-User-ID") != "user-1" {
				t.Fatalf("missing user header")
			}
			httpx.JSON(w, http.StatusOK, []Activity{{ID: "activity-1", UserID: "user-1", Type: "exercise", Title: "Move Your Body", XP: 40, Stat: "strength", OccurredAt: time.Now()}})
		case r.URL.Path == "/activities" && r.Method == http.MethodPost:
			if r.Header.Get("X-User-ID") != "user-1" {
				t.Fatalf("missing create user header")
			}
			httpx.JSON(w, http.StatusCreated, Activity{ID: "activity-2", UserID: "user-1", Type: "exercise", Title: "Move Your Body", XP: 40, Stat: "strength", OccurredAt: time.Now()})
		default:
			http.NotFound(w, r)
		}
	}))
	defer activityServer.Close()

	progressServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/progress/user-1" && r.Method == http.MethodGet:
			httpx.JSON(w, http.StatusOK, Progress{UserID: "user-1", Level: 1, TotalXP: 0, NextLevelXP: 100, Stats: map[string]int{}})
		case r.URL.Path == "/progress/award" && r.Method == http.MethodPost:
			var award map[string]any
			if err := json.NewDecoder(r.Body).Decode(&award); err != nil {
				t.Fatalf("decode award: %v", err)
			}
			if award["userId"] != "user-1" || award["activityId"] != "activity-2" {
				t.Fatalf("unexpected award: %+v", award)
			}
			httpx.JSON(w, http.StatusOK, Progress{UserID: "user-1", Level: 1, TotalXP: 40, CurrentLevelXP: 40, NextLevelXP: 100, Stats: map[string]int{"strength": 40}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer progressServer.Close()

	service := NewService(
		NewClient(identityServer.URL, time.Second),
		NewClient(activityServer.URL, time.Second),
		NewClient(progressServer.URL, time.Second),
		secret,
	)

	auth, err := service.Register(t.Context(), map[string]string{"email": "demo@example.com"})
	if err != nil || auth.Token != token {
		t.Fatalf("unexpected register result: %+v err=%v", auth, err)
	}
	auth, err = service.Login(t.Context(), map[string]string{"email": "demo@example.com"})
	if err != nil || auth.User.ID != "user-1" {
		t.Fatalf("unexpected login result: %+v err=%v", auth, err)
	}

	dashboard, err := service.Dashboard(t.Context(), token)
	if err != nil {
		t.Fatalf("Dashboard returned error: %v", err)
	}
	if dashboard.User.ID != "user-1" || len(dashboard.Activities) != 1 || len(dashboard.Rules) != 1 {
		t.Fatalf("unexpected dashboard: %+v", dashboard)
	}

	dashboard, err = service.CreateActivity(t.Context(), token, map[string]string{"type": "exercise"})
	if err != nil {
		t.Fatalf("CreateActivity returned error: %v", err)
	}
	if dashboard.Progress.TotalXP != 40 || len(dashboard.Activities) != 1 {
		t.Fatalf("unexpected create dashboard: %+v", dashboard)
	}

	if _, err := service.Me(t.Context(), token); err != nil {
		t.Fatalf("Me returned error: %v", err)
	}
	if _, err := service.Dashboard(t.Context(), "bad-token"); err == nil {
		t.Fatal("expected invalid token dashboard error")
	}
	if _, err := service.CreateActivity(t.Context(), "bad-token", nil); err == nil {
		t.Fatal("expected invalid token create activity error")
	}
}

func TestServicePropagatesDownstreamErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer server.Close()

	token, err := httpx.IssueToken("secret", "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	service := NewService(NewClient(server.URL, time.Second), NewClient(server.URL, time.Second), NewClient(server.URL, time.Second), "secret")

	if _, err := service.Me(t.Context(), token); err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected downstream me error, got %v", err)
	}
	if _, err := service.ActivityRules(t.Context()); err == nil {
		t.Fatal("expected activity rules downstream error")
	}
	if _, err := service.Activities(t.Context(), "user-1"); err == nil {
		t.Fatal("expected activities downstream error")
	}
	if _, err := service.Progress(t.Context(), "user-1"); err == nil {
		t.Fatal("expected progress downstream error")
	}
}

func TestDashboardPropagatesEachDownstreamError(t *testing.T) {
	token, err := httpx.IssueToken("secret", "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	okIdentity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"})
	}))
	defer okIdentity.Close()
	okProgress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, Progress{UserID: "user-1", Level: 1, NextLevelXP: 100, Stats: map[string]int{}})
	}))
	defer okProgress.Close()
	okActivity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/activities" {
			httpx.JSON(w, http.StatusOK, []Activity{})
			return
		}
		httpx.JSON(w, http.StatusOK, []ActivityRule{{Type: "exercise"}})
	}))
	defer okActivity.Close()
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer fail.Close()

	tests := []struct {
		name     string
		identity string
		progress string
		activity string
	}{
		{name: "me", identity: fail.URL, progress: okProgress.URL, activity: okActivity.URL},
		{name: "progress", identity: okIdentity.URL, progress: fail.URL, activity: okActivity.URL},
		{name: "activities", identity: okIdentity.URL, progress: okProgress.URL, activity: fail.URL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(NewClient(tt.identity, time.Second), NewClient(tt.activity, time.Second), NewClient(tt.progress, time.Second), "secret")
			if _, err := service.Dashboard(t.Context(), token); err == nil {
				t.Fatal("expected dashboard downstream error")
			}
		})
	}
}

func TestCreateActivityPropagatesEachDownstreamError(t *testing.T) {
	token, err := httpx.IssueToken("secret", "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	okIdentity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"})
	}))
	defer okIdentity.Close()
	okProgress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, Progress{UserID: "user-1", Level: 1, TotalXP: 40, NextLevelXP: 100, Stats: map[string]int{}})
	}))
	defer okProgress.Close()
	okActivity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			httpx.JSON(w, http.StatusCreated, Activity{ID: "activity-1", UserID: "user-1", XP: 40, Stat: "strength", OccurredAt: time.Now()})
			return
		}
		if r.URL.Path == "/activities" {
			httpx.JSON(w, http.StatusOK, []Activity{})
			return
		}
		httpx.JSON(w, http.StatusOK, []ActivityRule{{Type: "exercise"}})
	}))
	defer okActivity.Close()
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer fail.Close()

	tests := []struct {
		name     string
		identity string
		progress string
		activity string
	}{
		{name: "me", identity: fail.URL, progress: okProgress.URL, activity: okActivity.URL},
		{name: "activity create", identity: okIdentity.URL, progress: okProgress.URL, activity: fail.URL},
		{name: "award", identity: okIdentity.URL, progress: fail.URL, activity: okActivity.URL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(NewClient(tt.identity, time.Second), NewClient(tt.activity, time.Second), NewClient(tt.progress, time.Second), "secret")
			if _, err := service.CreateActivity(t.Context(), token, map[string]string{"type": "exercise"}); err == nil {
				t.Fatal("expected create activity downstream error")
			}
		})
	}
}
