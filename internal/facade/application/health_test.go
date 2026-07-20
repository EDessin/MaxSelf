package application

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

func TestHealthCandidatesBuildSyncedQuestClaims(t *testing.T) {
	now := time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC)
	pointsByType := map[string][]HealthDataPoint{
		"exercise": {
			{
				Name: "run-1",
				Exercise: &HealthExercise{
					Interval:       HealthInterval{StartTime: now.Add(-45 * time.Minute).Format(time.RFC3339), EndTime: now.Format(time.RFC3339)},
					ExerciseType:   "RUNNING",
					MetricsSummary: HealthMetricsSummary{DistanceMillimeters: 5000000},
				},
			},
			{
				DataPointName: "strength-1",
				Exercise: &HealthExercise{
					Interval:       HealthInterval{StartTime: now.Add(-2 * time.Hour).Format(time.RFC3339), EndTime: now.Add(-90 * time.Minute).Format(time.RFC3339)},
					ExerciseType:   "STRENGTH_TRAINING",
					ActiveDuration: "20m",
				},
			},
			{
				Name: "yoga-1",
				Exercise: &HealthExercise{
					Interval:       HealthInterval{StartTime: now.Add(-4 * time.Hour).Format(time.RFC3339), EndTime: now.Add(-3 * time.Hour).Format(time.RFC3339)},
					ExerciseType:   "YOGA",
					ActiveDuration: "15m",
				},
			},
			{
				Name: "short-run",
				Exercise: &HealthExercise{
					Interval:     HealthInterval{StartTime: now.Add(-5 * time.Minute).Format(time.RFC3339), EndTime: now.Format(time.RFC3339)},
					ExerciseType: "RUNNING",
				},
			},
			{
				Name: "unknown",
				Exercise: &HealthExercise{
					Interval:       HealthInterval{StartTime: now.Add(-30 * time.Minute).Format(time.RFC3339), EndTime: now.Format(time.RFC3339)},
					ExerciseType:   "FISHING",
					ActiveDuration: "30m",
				},
			},
		},
		"steps": {
			{
				Name: "steps-morning",
				Steps: &HealthSteps{
					Interval: HealthInterval{
						StartTime:      now.Add(-2 * time.Hour).Format(time.RFC3339),
						EndTime:        now.Add(-2*time.Hour + time.Minute).Format(time.RFC3339),
						CivilStartTime: HealthCivilDateTime{Date: HealthCivilDate{Year: 2026, Month: 7, Day: 17}},
					},
					Count: "3500",
				},
			},
			{
				Name: "steps-evening",
				Steps: &HealthSteps{
					Interval: HealthInterval{
						StartTime:      now.Add(8 * time.Hour).Format(time.RFC3339),
						EndTime:        now.Add(8*time.Hour + time.Minute).Format(time.RFC3339),
						CivilStartTime: HealthCivilDateTime{Date: HealthCivilDate{Year: 2026, Month: 7, Day: 17}},
					},
					Count: "7000",
				},
			},
			{
				Name: "steps-short-day",
				Steps: &HealthSteps{
					Interval: HealthInterval{
						StartTime:      now.Add(24 * time.Hour).Format(time.RFC3339),
						CivilStartTime: HealthCivilDateTime{Date: HealthCivilDate{Year: 2026, Month: 7, Day: 18}},
					},
					Count: "5999",
				},
			},
		},
		"sleep": {
			{
				Name: "sleep-1",
				Sleep: &HealthSleep{
					Interval: HealthInterval{StartTime: now.Add(-10 * time.Hour).Format(time.RFC3339), EndTime: now.Add(-2 * time.Hour).Format(time.RFC3339)},
					Summary:  HealthSleepSummary{MinutesAsleep: "450"},
				},
			},
			{
				Name: "nap",
				Sleep: &HealthSleep{
					Interval: HealthInterval{StartTime: now.Add(-12 * time.Hour).Format(time.RFC3339), EndTime: now.Add(-11 * time.Hour).Format(time.RFC3339)},
					Summary:  HealthSleepSummary{MinutesAsleep: "60"},
				},
			},
		},
		"hydration-log": {
			{
				Name: "hydration-1",
				HydrationLog: &HealthHydration{
					Interval:       HealthInterval{StartTime: now.Add(-time.Hour).Format(time.RFC3339), EndTime: now.Format(time.RFC3339)},
					AmountConsumed: HealthVolume{Milliliters: 700},
				},
			},
			{
				Name: "hydration-2",
				HydrationLog: &HealthHydration{
					Interval:       HealthInterval{StartTime: now.Add(-time.Hour).Format(time.RFC3339), EndTime: now.Format(time.RFC3339)},
					AmountConsumed: HealthVolume{Milliliters: 900},
				},
			},
		},
		"nutrition-log": {
			{
				Name: "meal-1",
				NutritionLog: &HealthNutrition{
					Interval: HealthInterval{StartTime: now.Add(-time.Hour).Format(time.RFC3339), EndTime: now.Format(time.RFC3339)},
				},
			},
		},
		"body-fat": {
			{
				Name:    "body-fat-1",
				BodyFat: &HealthBodyFat{SampleTime: HealthSampleTime{PhysicalTime: now.Format(time.RFC3339)}, Percentage: 22.5},
			},
			{
				Name:    "body-fat-2",
				BodyFat: &HealthBodyFat{SampleTime: HealthSampleTime{PhysicalTime: now.Add(24 * time.Hour).Format(time.RFC3339)}, Percentage: 21.0},
			},
		},
		"weight": {
			{
				Name:   "weight-1",
				Weight: &HealthWeight{SampleTime: HealthSampleTime{PhysicalTime: now.Format(time.RFC3339)}, WeightGrams: 70000},
			},
		},
	}

	candidates := healthCandidates("user-1", pointsByType)
	if len(candidates) != 13 {
		t.Fatalf("expected 13 candidates, got %d: %+v", len(candidates), candidates)
	}
	counts := map[string]int{}
	for _, candidate := range candidates {
		counts[candidate.Type]++
		if candidate.UserID != "user-1" ||
			candidate.Source != QuestClaimSourceGoogleHealth ||
			candidate.Status != QuestClaimStatusPending ||
			candidate.XP == 0 ||
			candidate.QuestDate == "" ||
			candidate.Evidence == "" {
			t.Fatalf("candidate missing required fields: %+v", candidate)
		}
	}
	expected := map[string]int{
		"cardio":             1,
		"daily_steps_bronze": 1,
		"daily_steps_silver": 1,
		"daily_steps_gold":   1,
		"exercise":           1,
		"recovery":           1,
		"sleep":              1,
		"hydration_bronze":   1,
		"hydration_silver":   1,
		"hydration_gold":     1,
		"healthy_meal":       1,
		"scale_measurement":  2,
	}
	for claimType, count := range expected {
		if counts[claimType] != count {
			t.Fatalf("unexpected count for %s: got %d counts=%+v candidates=%+v", claimType, counts[claimType], counts, candidates)
		}
	}
	if !containsEvidence(candidates, "Running · 45 min · 5.0 km") ||
		!containsEvidence(candidates, "7 hours 30 minutes asleep") ||
		!containsEvidence(candidates, "10500 steps") ||
		!containsEvidence(candidates, "1600 ml hydration logged") ||
		!containsEvidence(candidates, "Weight 70.0 kg, body fat 22.5%") ||
		!containsEvidence(candidates, "Body fat 21.0%") {
		t.Fatalf("expected evidence strings not found: %+v", candidates)
	}
}

func TestQuestTierClaimsUnlockSequentially(t *testing.T) {
	questDate := "2026-07-17"
	occurredAt := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	claims := []QuestClaim{
		newQuestClaim("user-1", "daily_steps_bronze", QuestClaimSourceGoogleHealth, "steps-bronze", occurredAt, "10500 steps"),
		newQuestClaim("user-1", "daily_steps_silver", QuestClaimSourceGoogleHealth, "steps-silver", occurredAt, "10500 steps"),
		newQuestClaim("user-1", "daily_steps_gold", QuestClaimSourceGoogleHealth, "steps-gold", occurredAt, "10500 steps"),
	}
	for index := range claims {
		claims[index].QuestDate = questDate
	}

	claimable := claimableQuestClaims(claims)
	if len(claimable) != 1 || claimable[0].Type != "daily_steps_bronze" {
		t.Fatalf("expected only bronze to be claimable, got %+v", claimable)
	}
	if questClaimIsClaimable(claims[1], claims) {
		t.Fatal("silver should be locked until bronze is claimed")
	}

	claims[0].Status = QuestClaimStatusClaimed
	claimable = claimableQuestClaims(claims)
	if len(claimable) != 1 || claimable[0].Type != "daily_steps_silver" || claimable[0].XP != 30 {
		t.Fatalf("expected silver to unlock with quest-rule XP, got %+v", claimable)
	}

	claims[1].Status = QuestClaimStatusClaimed
	claimable = claimableQuestClaims(claims)
	if len(claimable) != 1 || claimable[0].Type != "daily_steps_gold" || claimable[0].Title != "Daily Steps — Gold" {
		t.Fatalf("expected gold to unlock with quest-rule title, got %+v", claimable)
	}
}

func TestHealthHelpers(t *testing.T) {
	start := time.Date(2026, 7, 17, 0, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	queries := healthQueries(start, start.Add(2*time.Hour))
	if len(queries) != 7 || queries[0].dataType != "exercise" || queries[0].filter != `exercise.interval.civil_start_time >= "2026-07-17T00:00:00"` {
		t.Fatalf("unexpected queries: %+v", queries)
	}
	expectedFilters := map[string]string{
		"steps":         `steps.interval.civil_start_time >= "2026-07-17T00:00:00"`,
		"sleep":         `sleep.interval.end_time >= "2026-07-16T22:00:00Z"`,
		"hydration-log": `hydration_log.interval.civil_start_time >= "2026-07-17T00:00:00"`,
		"nutrition-log": `nutrition_log.interval.civil_start_time >= "2026-07-17T00:00:00"`,
		"weight":        `weight.sample_time.physical_time >= "2026-07-16T22:00:00Z"`,
		"body-fat":      `body_fat.sample_time.physical_time >= "2026-07-16T22:00:00Z"`,
	}
	for _, query := range queries[1:] {
		if expectedFilters[query.dataType] != query.filter {
			t.Fatalf("unexpected filter for %s: %s", query.dataType, query.filter)
		}
	}
	start = start.Add(14 * time.Hour)
	queries = healthQueries(start, start.Add(2*time.Hour))
	if queries[1].filter != `steps.interval.civil_start_time >= "2026-07-17T00:00:00"` {
		t.Fatalf("steps filter should start at the civil day boundary, got %s", queries[1].filter)
	}
	if questTypeForExercise("indoor_cycling") != "cardio" ||
		questTypeForExercise("resistance") != "exercise" ||
		questTypeForExercise("mobility") != "recovery" ||
		questTypeForExercise("fishing") != "" {
		t.Fatal("exercise quest type mapping regressed")
	}
	duration := exerciseDuration(HealthExercise{
		ActiveDuration: "not-a-duration",
		Interval:       HealthInterval{StartTime: start.Format(time.RFC3339), EndTime: start.Add(20 * time.Minute).Format(time.RFC3339)},
	})
	if duration != 20*time.Minute || exerciseDuration(HealthExercise{}) != 0 {
		t.Fatalf("unexpected exercise durations: %s", duration)
	}
	if displayExerciseType("") != "Exercise" || displayExerciseType("STRENGTH_TRAINING") != "Strength Training" {
		t.Fatal("exercise type display regressed")
	}
	if sleepEvidence(450) != "7 hours 30 minutes asleep" || sleepEvidence(421) != "7 hours 1 minute asleep" {
		t.Fatal("sleep evidence display regressed")
	}
	if (HealthDataPoint{DataPointName: "data-point", Name: "name"}).ID() != "data-point" ||
		(HealthDataPoint{Name: "name"}).ID() != "name" ||
		(HealthDataPoint{}).ID() == "" {
		t.Fatal("data point ID selection regressed")
	}
	if !parseHealthTime("not-a-time").IsZero() || !sampleTime(HealthSampleTime{}).IsZero() {
		t.Fatal("health time parsing regressed")
	}
	fallback := ruleForType("custom")
	if fallback.Type != "custom" || fallback.XP != 0 || fallback.Stat != "" {
		t.Fatalf("unexpected fallback rule: %+v", fallback)
	}
	for _, rule := range localActivityRules() {
		if rule.Color != activityRuleCategoryColor(rule.Stat) {
			t.Fatalf("expected local rule %s to use %s category color %s, got %s", rule.Type, rule.Stat, activityRuleCategoryColor(rule.Stat), rule.Color)
		}
	}
	recoveryRule := ruleForType("recovery")
	if recoveryRule.Title != "Recovery Ritual" || recoveryRule.Color != "#6366f1" {
		t.Fatalf("expected Recovery Ritual to use recovery category color, got %+v", recoveryRule)
	}
}

func TestHealthCandidatesIgnoreIncompleteData(t *testing.T) {
	now := time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC)
	candidates := healthCandidates("user-1", map[string][]HealthDataPoint{
		"exercise": {
			{},
			{Exercise: &HealthExercise{ExerciseType: "RUNNING"}},
			{Exercise: &HealthExercise{ExerciseType: "RUNNING", Interval: HealthInterval{StartTime: now.Format(time.RFC3339)}}},
		},
		"sleep": {
			{},
			{Sleep: &HealthSleep{Summary: HealthSleepSummary{MinutesAsleep: "not-a-number"}}},
			{Sleep: &HealthSleep{Summary: HealthSleepSummary{MinutesAsleep: "450"}}},
		},
		"steps": {
			{},
			{Steps: &HealthSteps{Count: "not-a-number"}},
			{Steps: &HealthSteps{Count: "0", Interval: HealthInterval{StartTime: now.Format(time.RFC3339)}}},
			{Steps: &HealthSteps{Count: "6000"}},
			{Steps: &HealthSteps{Count: "5999", Interval: HealthInterval{StartTime: now.Format(time.RFC3339)}}},
		},
		"hydration-log": {
			{},
			{HydrationLog: &HealthHydration{AmountConsumed: HealthVolume{Milliliters: 300}}},
		},
		"nutrition-log": {
			{},
			{NutritionLog: &HealthNutrition{}},
		},
		"body-fat": {
			{},
			{BodyFat: &HealthBodyFat{Percentage: 22}},
		},
		"weight": {
			{},
			{Weight: &HealthWeight{WeightGrams: 70000}},
		},
	})
	if len(candidates) != 0 {
		t.Fatalf("expected incomplete health data to be ignored, got %+v", candidates)
	}
}

func TestServiceGoogleHealthErrorPaths(t *testing.T) {
	secret := "secret"
	token, err := httpx.IssueToken(secret, "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	unconfigured := NewService(NewClient("http://identity.invalid", time.Second), NewClient("http://activity.invalid", time.Second), NewClient("http://progress.invalid", time.Second), secret)
	if status := unconfigured.GoogleHealthStatus(context.Background(), "user-1"); status.Connected || status.PendingClaims != 0 {
		t.Fatalf("unexpected unconfigured health status: %+v", status)
	}
	if claims := unconfigured.PendingQuestClaims(context.Background(), "user-1"); claims != nil {
		t.Fatalf("expected nil unconfigured pending claims, got %+v", claims)
	}
	if _, err := unconfigured.StartGoogleHealthConnect(context.Background(), token); !errors.Is(err, ErrGoogleHealthNotConfigured) {
		t.Fatalf("expected unconfigured connect error, got %v", err)
	}
	if err := unconfigured.CompleteGoogleHealthConnect(context.Background(), "state", "code"); !errors.Is(err, ErrGoogleHealthNotConfigured) {
		t.Fatalf("expected unconfigured callback error, got %v", err)
	}
	if _, err := unconfigured.SyncGoogleHealth(context.Background(), token); !errors.Is(err, ErrGoogleHealthNotConfigured) {
		t.Fatalf("expected unconfigured sync error, got %v", err)
	}
	if _, err := unconfigured.ClaimQuest(context.Background(), token, "claim-1"); !errors.Is(err, ErrGoogleHealthNotConfigured) {
		t.Fatalf("expected unconfigured claim error, got %v", err)
	}

	repo := newFakeIntegrationRepository()
	service := NewServiceWithIntegrations(
		NewClient("http://identity.invalid", time.Second),
		NewClient("http://activity.invalid", time.Second),
		NewClient("http://progress.invalid", time.Second),
		secret,
		repo,
		errorGoogleHealthClient{reconcileErr: errors.New("sync failed")},
	)
	if _, err := service.StartGoogleHealthConnect(context.Background(), "bad-token"); err == nil {
		t.Fatal("expected bad connect token error")
	}
	if err := service.CompleteGoogleHealthConnect(context.Background(), "missing-state", "code"); err == nil {
		t.Fatal("expected missing auth state error")
	}
	if _, err := service.SyncGoogleHealth(context.Background(), token); !errors.Is(err, ErrGoogleHealthNotConnected) {
		t.Fatalf("expected not connected sync error, got %v", err)
	}
	repo.connections["user-1"] = HealthConnection{UserID: "user-1", RefreshToken: "refresh"}
	if _, err := service.SyncGoogleHealth(context.Background(), token); err == nil || !strings.Contains(err.Error(), "sync failed") {
		t.Fatalf("expected reconcile error, got %v", err)
	}
	if _, err := service.ClaimQuest(context.Background(), token, "missing-claim"); !errors.Is(err, ErrQuestClaimNotFound) {
		t.Fatalf("expected missing claim error, got %v", err)
	}
	claimed := newQuestClaim("user-1", "cardio", QuestClaimSourceGoogleHealth, "run-1", time.Now(), "Run")
	claimed.Status = QuestClaimStatusClaimed
	repo.claims[claimed.ID] = claimed
	if _, err := service.ClaimQuest(context.Background(), token, claimed.ID); !errors.Is(err, ErrQuestClaimAlreadyClaimed) {
		t.Fatalf("expected already claimed error, got %v", err)
	}
}

func TestServiceCreateWaistToHeightClaim(t *testing.T) {
	secret := "secret"
	token, err := httpx.IssueToken(secret, "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"})
	}))
	defer identityServer.Close()
	activityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/activity-types":
			httpx.JSON(w, http.StatusOK, localActivityRules())
		case "/activities":
			httpx.JSON(w, http.StatusOK, []Activity{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer activityServer.Close()
	progressServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, Progress{UserID: "user-1", Level: 1, NextLevelXP: 100, Stats: map[string]int{}})
	}))
	defer progressServer.Close()

	service := NewServiceWithIntegrations(
		NewClient(identityServer.URL, time.Second),
		NewClient(activityServer.URL, time.Second),
		NewClient(progressServer.URL, time.Second),
		secret,
		newFakeIntegrationRepository(),
		nil,
	)
	measuredAt := time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC)
	result, err := service.CreateWaistToHeightClaim(context.Background(), token, WaistToHeightRequest{
		WaistCentimeters:  80,
		HeightCentimeters: 180,
		MeasuredAt:        &measuredAt,
	})
	if err != nil {
		t.Fatalf("CreateWaistToHeightClaim returned error: %v", err)
	}
	if result.CreatedClaims != 1 || len(result.PendingClaims) != 1 || result.PendingClaims[0].Type != "waist_to_height_ratio" {
		t.Fatalf("unexpected waist result: %+v", result)
	}
	if !strings.Contains(result.PendingClaims[0].Evidence, "ratio 0.44") {
		t.Fatalf("unexpected waist evidence: %s", result.PendingClaims[0].Evidence)
	}
	result, err = service.CreateWaistToHeightClaim(context.Background(), token, WaistToHeightRequest{
		WaistCentimeters:  81,
		HeightCentimeters: 180,
		MeasuredAt:        &measuredAt,
	})
	if err != nil {
		t.Fatalf("duplicate CreateWaistToHeightClaim returned error: %v", err)
	}
	if result.CreatedClaims != 0 {
		t.Fatalf("expected duplicate waist claim not to be created: %+v", result)
	}
	if _, err := service.CreateWaistToHeightClaim(context.Background(), token, WaistToHeightRequest{}); err == nil {
		t.Fatal("expected invalid waist request error")
	}

	unconfigured := NewService(
		NewClient(identityServer.URL, time.Second),
		NewClient(activityServer.URL, time.Second),
		NewClient(progressServer.URL, time.Second),
		secret,
	)
	if _, err := unconfigured.CreateWaistToHeightClaim(context.Background(), token, WaistToHeightRequest{WaistCentimeters: 80, HeightCentimeters: 180}); !errors.Is(err, ErrGoogleHealthNotConfigured) {
		t.Fatalf("expected unconfigured waist error, got %v", err)
	}
}

func containsEvidence(candidates []QuestClaim, evidence string) bool {
	for _, candidate := range candidates {
		if candidate.Evidence == evidence {
			return true
		}
	}
	return false
}

type errorGoogleHealthClient struct {
	exchangeErr  error
	reconcileErr error
}

func (c errorGoogleHealthClient) AuthCodeURL(state string) string {
	return "https://google.example/health?state=" + state
}

func (c errorGoogleHealthClient) ExchangeCode(context.Context, string) (HealthConnection, error) {
	if c.exchangeErr != nil {
		return HealthConnection{}, c.exchangeErr
	}
	return HealthConnection{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, nil
}

func (c errorGoogleHealthClient) Reconcile(context.Context, HealthConnection, string, string) (HealthConnection, []HealthDataPoint, error) {
	if c.reconcileErr != nil {
		return HealthConnection{}, nil, c.reconcileErr
	}
	return HealthConnection{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, nil, nil
}
