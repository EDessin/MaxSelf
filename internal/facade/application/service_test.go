package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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
			httpx.JSON(w, http.StatusOK, []ActivityRule{{Type: "exercise", Title: "Strength Session", XP: 40, Stat: "strength"}})
		case r.URL.Path == "/activities" && r.Method == http.MethodGet:
			if r.Header.Get("X-User-ID") != "user-1" {
				t.Fatalf("missing user header")
			}
			httpx.JSON(w, http.StatusOK, []Activity{{ID: "activity-1", UserID: "user-1", Type: "exercise", Title: "Strength Session", XP: 40, Stat: "strength", OccurredAt: time.Now()}})
		case r.URL.Path == "/activities" && r.Method == http.MethodPost:
			if r.Header.Get("X-User-ID") != "user-1" {
				t.Fatalf("missing create user header")
			}
			httpx.JSON(w, http.StatusCreated, Activity{ID: "activity-2", UserID: "user-1", Type: "exercise", Title: "Strength Session", XP: 40, Stat: "strength", OccurredAt: time.Now()})
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

	if _, err := service.Me(t.Context(), token); err != nil {
		t.Fatalf("Me returned error: %v", err)
	}
	if _, err := service.Dashboard(t.Context(), "bad-token"); err == nil {
		t.Fatal("expected invalid token dashboard error")
	}
	if _, err := service.CreateActivity(t.Context(), token, nil); err == nil || !strings.Contains(err.Error(), "manual XP claims are disabled") {
		t.Fatalf("expected manual claim disabled error, got %v", err)
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

func TestServiceGoogleHealthSyncAndClaim(t *testing.T) {
	secret := "secret"
	token, err := httpx.IssueToken(secret, "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"})
	}))
	defer identityServer.Close()

	var createdType string
	activityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/activity-types":
			httpx.JSON(w, http.StatusOK, []ActivityRule{{Type: "cardio", Title: "Cardio Session", XP: 30, Stat: "cardio"}})
		case r.URL.Path == "/activities" && r.Method == http.MethodGet:
			httpx.JSON(w, http.StatusOK, []Activity{})
		case r.URL.Path == "/activities" && r.Method == http.MethodPost:
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode activity: %v", err)
			}
			createdType, _ = req["type"].(string)
			httpx.JSON(w, http.StatusCreated, Activity{ID: "activity-claim", UserID: "user-1", Type: "cardio", Title: "Cardio Session", XP: 30, Stat: "cardio", OccurredAt: time.Now()})
		default:
			http.NotFound(w, r)
		}
	}))
	defer activityServer.Close()

	progressTotal := 0
	progressServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/progress/user-1":
			httpx.JSON(w, http.StatusOK, Progress{UserID: "user-1", Level: 1, TotalXP: progressTotal, CurrentLevelXP: progressTotal, NextLevelXP: 100, Stats: map[string]int{"cardio": progressTotal}})
		case "/progress/award":
			var award map[string]any
			if err := json.NewDecoder(r.Body).Decode(&award); err != nil {
				t.Fatalf("decode award: %v", err)
			}
			if award["activityId"] != "activity-claim" || award["stat"] != "cardio" {
				t.Fatalf("unexpected award: %+v", award)
			}
			progressTotal = 30
			httpx.JSON(w, http.StatusOK, Progress{UserID: "user-1", Level: 1, TotalXP: progressTotal, CurrentLevelXP: progressTotal, NextLevelXP: 100, Stats: map[string]int{"cardio": progressTotal}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer progressServer.Close()

	now := time.Now().UTC()
	repo := newFakeIntegrationRepository()
	health := &fakeGoogleHealthClient{
		points: map[string][]HealthDataPoint{
			"exercise": {{
				Name: "users/me/dataTypes/exercise/dataPoints/run-1",
				Exercise: &HealthExercise{
					Interval:     HealthInterval{StartTime: now.Add(-35 * time.Minute).Format(time.RFC3339), EndTime: now.Format(time.RFC3339)},
					ExerciseType: "RUNNING",
				},
			}},
		},
	}
	service := NewServiceWithIntegrations(
		NewClient(identityServer.URL, time.Second),
		NewClient(activityServer.URL, time.Second),
		NewClient(progressServer.URL, time.Second),
		secret,
		repo,
		health,
	)

	connect, err := service.StartGoogleHealthConnect(t.Context(), token)
	if err != nil {
		t.Fatalf("StartGoogleHealthConnect returned error: %v", err)
	}
	parsed, err := url.Parse(connect.URL)
	if err != nil {
		t.Fatalf("parse connect URL: %v", err)
	}
	if err := service.CompleteGoogleHealthConnect(t.Context(), parsed.Query().Get("state"), "code"); err != nil {
		t.Fatalf("CompleteGoogleHealthConnect returned error: %v", err)
	}

	syncResult, err := service.SyncGoogleHealth(t.Context(), token)
	if err != nil {
		t.Fatalf("SyncGoogleHealth returned error: %v", err)
	}
	if syncResult.CreatedClaims != 1 || len(syncResult.PendingClaims) != 1 || syncResult.PendingClaims[0].Type != "cardio" {
		t.Fatalf("unexpected sync result: %+v", syncResult)
	}

	dashboard, err := service.ClaimQuest(t.Context(), token, syncResult.PendingClaims[0].ID)
	if err != nil {
		t.Fatalf("ClaimQuest returned error: %v", err)
	}
	if createdType != "cardio" || dashboard.Progress.TotalXP != 30 || len(dashboard.QuestClaims) != 0 {
		t.Fatalf("unexpected claim result createdType=%s dashboard=%+v", createdType, dashboard)
	}
}

type fakeGoogleHealthClient struct {
	points map[string][]HealthDataPoint
}

func (c *fakeGoogleHealthClient) AuthCodeURL(state string) string {
	return "https://google.example/health?state=" + url.QueryEscape(state)
}

func (c *fakeGoogleHealthClient) ExchangeCode(_ context.Context, _ string) (HealthConnection, error) {
	return HealthConnection{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, nil
}

func (c *fakeGoogleHealthClient) Reconcile(_ context.Context, connection HealthConnection, dataType string, _ string) (HealthConnection, []HealthDataPoint, error) {
	return connection, c.points[dataType], nil
}

type fakeIntegrationRepository struct {
	states      map[string]HealthAuthState
	connections map[string]HealthConnection
	claims      map[string]QuestClaim
}

func newFakeIntegrationRepository() *fakeIntegrationRepository {
	return &fakeIntegrationRepository{
		states:      map[string]HealthAuthState{},
		connections: map[string]HealthConnection{},
		claims:      map[string]QuestClaim{},
	}
}

func (r *fakeIntegrationRepository) SaveHealthAuthState(_ context.Context, state HealthAuthState) error {
	r.states[state.State] = state
	return nil
}

func (r *fakeIntegrationRepository) ConsumeHealthAuthState(_ context.Context, state string, now time.Time) (HealthAuthState, error) {
	authState, ok := r.states[state]
	if !ok || authState.UsedAt != nil || !authState.ExpiresAt.After(now) {
		return HealthAuthState{}, ErrQuestClaimNotFound
	}
	authState.UsedAt = &now
	r.states[state] = authState
	return authState, nil
}

func (r *fakeIntegrationRepository) SaveHealthConnection(_ context.Context, connection HealthConnection) error {
	r.connections[connection.UserID] = connection
	return nil
}

func (r *fakeIntegrationRepository) GetHealthConnection(_ context.Context, userID string) (HealthConnection, error) {
	connection, ok := r.connections[userID]
	if !ok {
		return HealthConnection{}, ErrGoogleHealthNotConnected
	}
	return connection, nil
}

func (r *fakeIntegrationRepository) UpdateHealthConnectionSync(_ context.Context, userID string, syncedAt time.Time) error {
	connection := r.connections[userID]
	connection.LastSyncedAt = &syncedAt
	r.connections[userID] = connection
	return nil
}

func (r *fakeIntegrationRepository) UpsertQuestClaim(_ context.Context, claim QuestClaim) (QuestClaim, bool, error) {
	for _, existing := range r.claims {
		if existing.UserID == claim.UserID && existing.Type == claim.Type && existing.QuestDate == claim.QuestDate {
			return existing, false, nil
		}
	}
	r.claims[claim.ID] = claim
	return claim, true, nil
}

func (r *fakeIntegrationRepository) ListPendingQuestClaims(_ context.Context, userID string) ([]QuestClaim, error) {
	var claims []QuestClaim
	for _, claim := range r.claims {
		if claim.UserID == userID && claim.Status == QuestClaimStatusPending {
			claims = append(claims, claim)
		}
	}
	return claims, nil
}

func (r *fakeIntegrationRepository) CountPendingQuestClaims(_ context.Context, userID string) (int, error) {
	claims, err := r.ListPendingQuestClaims(context.Background(), userID)
	return len(claims), err
}

func (r *fakeIntegrationRepository) GetQuestClaim(_ context.Context, userID string, claimID string) (QuestClaim, error) {
	claim, ok := r.claims[claimID]
	if !ok || claim.UserID != userID {
		return QuestClaim{}, ErrQuestClaimNotFound
	}
	return claim, nil
}

func (r *fakeIntegrationRepository) MarkQuestClaimClaimed(_ context.Context, userID string, claimID string, activityID string, claimedAt time.Time) error {
	claim, ok := r.claims[claimID]
	if !ok || claim.UserID != userID || claim.Status != QuestClaimStatusPending {
		return ErrQuestClaimAlreadyClaimed
	}
	claim.Status = QuestClaimStatusClaimed
	claim.ActivityID = activityID
	claim.ClaimedAt = &claimedAt
	r.claims[claimID] = claim
	return nil
}
