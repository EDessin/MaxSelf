package application

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	QuestClaimStatusPending = "pending"
	QuestClaimStatusClaimed = "claimed"

	QuestClaimSourceGoogleHealth = "google_health"
	QuestClaimSourceManual       = "manual"

	sleepGoalHours = 7
)

var (
	ErrGoogleHealthNotConfigured = errors.New("google health integration is not configured")
	ErrGoogleHealthNotConnected  = errors.New("google health is not connected")
	ErrQuestClaimNotFound        = errors.New("quest claim not found")
	ErrQuestClaimAlreadyClaimed  = errors.New("quest claim already claimed")
	ErrQuestClaimLocked          = errors.New("complete the previous quest tier first")
)

type IntegrationRepository interface {
	SaveHealthAuthState(ctx context.Context, state HealthAuthState) error
	ConsumeHealthAuthState(ctx context.Context, state string, now time.Time) (HealthAuthState, error)
	SaveHealthConnection(ctx context.Context, connection HealthConnection) error
	GetHealthConnection(ctx context.Context, userID string) (HealthConnection, error)
	UpdateHealthConnectionSync(ctx context.Context, userID string, syncedAt time.Time) error
	UpsertQuestClaim(ctx context.Context, claim QuestClaim) (QuestClaim, bool, error)
	ListQuestClaims(ctx context.Context, userID string) ([]QuestClaim, error)
	ListPendingQuestClaims(ctx context.Context, userID string) ([]QuestClaim, error)
	CountPendingQuestClaims(ctx context.Context, userID string) (int, error)
	GetQuestClaim(ctx context.Context, userID string, claimID string) (QuestClaim, error)
	MarkQuestClaimClaimed(ctx context.Context, userID string, claimID string, activityID string, claimedAt time.Time) error
}

type GoogleHealthClient interface {
	AuthCodeURL(state string) string
	ExchangeCode(ctx context.Context, code string) (HealthConnection, error)
	Reconcile(ctx context.Context, connection HealthConnection, dataType string, filter string) (HealthConnection, []HealthDataPoint, error)
}

type HealthAuthState struct {
	State     string
	UserID    string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type HealthConnection struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	Expiry       time.Time
	LastSyncedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type HealthIntegrationStatus struct {
	Connected     bool       `json:"connected"`
	LastSyncedAt  *time.Time `json:"lastSyncedAt"`
	PendingClaims int        `json:"pendingClaims"`
}

type QuestClaim struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	XP         int        `json:"xp"`
	Stat       string     `json:"stat"`
	Source     string     `json:"source"`
	SourceID   string     `json:"sourceId"`
	Evidence   string     `json:"evidence"`
	OccurredAt time.Time  `json:"occurredAt"`
	QuestDate  string     `json:"questDate"`
	Status     string     `json:"status"`
	ActivityID string     `json:"activityId"`
	CreatedAt  time.Time  `json:"createdAt"`
	ClaimedAt  *time.Time `json:"claimedAt"`
}

type HealthSyncResult struct {
	CreatedClaims int          `json:"createdClaims"`
	PendingClaims []QuestClaim `json:"pendingClaims"`
	Dashboard     Dashboard    `json:"dashboard"`
}

type HealthConnectResult struct {
	URL string `json:"url"`
}

type WaistToHeightRequest struct {
	WaistCentimeters  float64    `json:"waistCentimeters"`
	HeightCentimeters float64    `json:"heightCentimeters"`
	MeasuredAt        *time.Time `json:"measuredAt"`
}

type HealthDataPoint struct {
	Name          string           `json:"name"`
	DataPointName string           `json:"dataPointName"`
	DataSource    HealthDataSource `json:"dataSource"`
	Exercise      *HealthExercise  `json:"exercise"`
	Steps         *HealthSteps     `json:"steps"`
	Sleep         *HealthSleep     `json:"sleep"`
	Weight        *HealthWeight    `json:"weight"`
	BodyFat       *HealthBodyFat   `json:"bodyFat"`
	HydrationLog  *HealthHydration `json:"hydrationLog"`
	NutritionLog  *HealthNutrition `json:"nutritionLog"`
}

type HealthDataSource struct {
	RecordingMethod string             `json:"recordingMethod"`
	Device          HealthSourceDevice `json:"device"`
	Platform        string             `json:"platform"`
}

type HealthSourceDevice struct {
	FormFactor  string `json:"formFactor"`
	DisplayName string `json:"displayName"`
}

type HealthInterval struct {
	StartTime      string              `json:"startTime"`
	EndTime        string              `json:"endTime"`
	CivilStartTime HealthCivilDateTime `json:"civilStartTime"`
	CivilEndTime   HealthCivilDateTime `json:"civilEndTime"`
}

type HealthCivilDateTime struct {
	Date HealthCivilDate `json:"date"`
	Time HealthCivilTime `json:"time"`
}

type HealthCivilDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type HealthCivilTime struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
	Nanos   int `json:"nanos"`
}

type HealthSampleTime struct {
	PhysicalTime string `json:"physicalTime"`
}

type HealthExercise struct {
	Interval       HealthInterval       `json:"interval"`
	ExerciseType   string               `json:"exerciseType"`
	ActiveDuration string               `json:"activeDuration"`
	MetricsSummary HealthMetricsSummary `json:"metricsSummary"`
}

type HealthSteps struct {
	Interval HealthInterval `json:"interval"`
	Count    string         `json:"count"`
}

type HealthMetricsSummary struct {
	DistanceMillimeters float64 `json:"distanceMillimeters"`
	CaloriesKcal        float64 `json:"caloriesKcal"`
	Steps               string  `json:"steps"`
}

type HealthSleep struct {
	Interval HealthInterval     `json:"interval"`
	Summary  HealthSleepSummary `json:"summary"`
}

type HealthSleepSummary struct {
	MinutesAsleep string `json:"minutesAsleep"`
}

type HealthWeight struct {
	SampleTime  HealthSampleTime `json:"sampleTime"`
	WeightGrams float64          `json:"weightGrams"`
}

type HealthBodyFat struct {
	SampleTime HealthSampleTime `json:"sampleTime"`
	Percentage float64          `json:"percentage"`
}

type HealthHydration struct {
	Interval       HealthInterval `json:"interval"`
	AmountConsumed HealthVolume   `json:"amountConsumed"`
}

type HealthVolume struct {
	Milliliters float64 `json:"milliliters"`
}

type HealthNutrition struct {
	Interval HealthInterval `json:"interval"`
}

func (s Service) GoogleHealthStatus(ctx context.Context, userID string) HealthIntegrationStatus {
	if s.integrations == nil {
		return HealthIntegrationStatus{}
	}
	status := HealthIntegrationStatus{}
	connection, err := s.integrations.GetHealthConnection(ctx, userID)
	if err == nil {
		status.Connected = connection.RefreshToken != ""
		status.LastSyncedAt = connection.LastSyncedAt
	}
	pending, err := s.pendingQuestClaims(ctx, userID)
	if err == nil {
		status.PendingClaims = len(pending)
	}
	return status
}

func (s Service) PendingQuestClaims(ctx context.Context, userID string) []QuestClaim {
	if s.integrations == nil {
		return nil
	}
	claims, err := s.pendingQuestClaims(ctx, userID)
	if err != nil {
		return nil
	}
	return claims
}

func (s Service) QuestClaimHistory(ctx context.Context, userID string) []QuestClaim {
	if s.integrations == nil {
		return nil
	}
	claims, err := s.integrations.ListQuestClaims(ctx, userID)
	if err != nil {
		return nil
	}
	for index := range claims {
		claims[index] = claimWithRule(claims[index])
	}
	sortQuestClaims(claims)
	return claims
}

func (s Service) pendingQuestClaims(ctx context.Context, userID string) ([]QuestClaim, error) {
	if s.integrations == nil {
		return nil, nil
	}
	claims, err := s.integrations.ListQuestClaims(ctx, userID)
	if err != nil {
		return nil, err
	}
	return claimableQuestClaims(claims), nil
}

func (s Service) StartGoogleHealthConnect(ctx context.Context, token string) (HealthConnectResult, error) {
	if s.integrations == nil || s.googleHealth == nil {
		return HealthConnectResult{}, ErrGoogleHealthNotConfigured
	}
	claims, err := parseClaims(s.jwtSecret, token)
	if err != nil {
		return HealthConnectResult{}, err
	}
	state := uuid.NewString()
	if err := s.integrations.SaveHealthAuthState(ctx, HealthAuthState{
		State:     state,
		UserID:    claims.UserID,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}); err != nil {
		return HealthConnectResult{}, err
	}
	return HealthConnectResult{URL: s.googleHealth.AuthCodeURL(state)}, nil
}

func (s Service) CompleteGoogleHealthConnect(ctx context.Context, state string, code string) error {
	if s.integrations == nil || s.googleHealth == nil {
		return ErrGoogleHealthNotConfigured
	}
	authState, err := s.integrations.ConsumeHealthAuthState(ctx, state, time.Now())
	if err != nil {
		log.Printf("google health connect callback state consume failed state=%s err=%v", state, err)
		return err
	}
	connection, err := s.googleHealth.ExchangeCode(ctx, code)
	if err != nil {
		log.Printf("google health connect callback exchange failed user_id=%s err=%v", authState.UserID, err)
		return err
	}
	connection.UserID = authState.UserID
	if err := s.integrations.SaveHealthConnection(ctx, connection); err != nil {
		log.Printf("google health connect save connection failed user_id=%s err=%v", authState.UserID, err)
		return err
	}
	log.Printf("google health connected user_id=%s scope=%q expiry=%s", authState.UserID, connection.Scope, connection.Expiry.Format(time.RFC3339))
	return nil
}

func (s Service) SyncGoogleHealth(ctx context.Context, token string) (HealthSyncResult, error) {
	claims, err := parseClaims(s.jwtSecret, token)
	if err != nil {
		return HealthSyncResult{}, err
	}
	if s.integrations == nil || s.googleHealth == nil {
		return HealthSyncResult{}, ErrGoogleHealthNotConfigured
	}
	connection, err := s.integrations.GetHealthConnection(ctx, claims.UserID)
	if err != nil || connection.RefreshToken == "" {
		log.Printf("google health sync blocked user_id=%s connected=false err=%v", claims.UserID, err)
		return HealthSyncResult{}, ErrGoogleHealthNotConnected
	}

	now := time.Now()
	start := startOfToday(now)
	if connection.LastSyncedAt != nil {
		start = connection.LastSyncedAt.Add(-6 * time.Hour)
	}
	oldest := now.AddDate(0, 0, -90)
	if start.Before(oldest) {
		start = oldest
	}

	log.Printf("google health sync started user_id=%s start=%s last_synced_at=%v", claims.UserID, start.UTC().Format(time.RFC3339), connection.LastSyncedAt)
	pointsByType := map[string][]HealthDataPoint{}
	for _, query := range healthQueries(start, now) {
		var points []HealthDataPoint
		connection, points, err = s.googleHealth.Reconcile(ctx, connection, query.dataType, query.filter)
		if err != nil {
			log.Printf("google health sync query failed user_id=%s data_type=%s filter=%q err=%v", claims.UserID, query.dataType, query.filter, err)
			return HealthSyncResult{}, err
		}
		log.Printf("google health sync query completed user_id=%s data_type=%s points=%d", claims.UserID, query.dataType, len(points))
		pointsByType[query.dataType] = points
	}
	if err := s.integrations.SaveHealthConnection(ctx, connection); err != nil {
		log.Printf("google health sync save connection failed user_id=%s err=%v", claims.UserID, err)
		return HealthSyncResult{}, err
	}

	candidates := healthCandidates(claims.UserID, pointsByType)
	log.Printf("google health sync candidates built user_id=%s candidates=%d", claims.UserID, len(candidates))
	created := 0
	for _, candidate := range candidates {
		_, inserted, err := s.integrations.UpsertQuestClaim(ctx, candidate)
		if err != nil {
			log.Printf("google health sync upsert claim failed user_id=%s type=%s quest_date=%s source_id=%s err=%v", claims.UserID, candidate.Type, candidate.QuestDate, candidate.SourceID, err)
			return HealthSyncResult{}, err
		}
		if inserted {
			created++
		}
	}
	if err := s.integrations.UpdateHealthConnectionSync(ctx, claims.UserID, now); err != nil {
		log.Printf("google health sync timestamp update failed user_id=%s err=%v", claims.UserID, err)
		return HealthSyncResult{}, err
	}

	pending, err := s.pendingQuestClaims(ctx, claims.UserID)
	if err != nil {
		return HealthSyncResult{}, err
	}
	dashboard, err := s.Dashboard(ctx, token)
	if err != nil {
		return HealthSyncResult{}, err
	}
	log.Printf("google health sync completed user_id=%s created_claims=%d pending_claims=%d", claims.UserID, created, len(pending))
	return HealthSyncResult{CreatedClaims: created, PendingClaims: pending, Dashboard: dashboard}, nil
}

func (s Service) CreateWaistToHeightClaim(ctx context.Context, token string, req WaistToHeightRequest) (HealthSyncResult, error) {
	claims, err := parseClaims(s.jwtSecret, token)
	if err != nil {
		return HealthSyncResult{}, err
	}
	if s.integrations == nil {
		return HealthSyncResult{}, ErrGoogleHealthNotConfigured
	}
	if req.WaistCentimeters <= 0 || req.HeightCentimeters <= 0 {
		return HealthSyncResult{}, errors.New("waist and height are required")
	}
	measuredAt := time.Now()
	if req.MeasuredAt != nil {
		measuredAt = *req.MeasuredAt
	}
	ratio := req.WaistCentimeters / req.HeightCentimeters
	candidate := newQuestClaim(
		claims.UserID,
		"waist_to_height_ratio",
		QuestClaimSourceManual,
		fmt.Sprintf("manual-waist-height-%s", dateKey(measuredAt)),
		measuredAt,
		fmt.Sprintf("Waist %.1f cm, height %.1f cm, ratio %.2f", req.WaistCentimeters, req.HeightCentimeters, ratio),
	)
	_, inserted, err := s.integrations.UpsertQuestClaim(ctx, candidate)
	if err != nil {
		return HealthSyncResult{}, err
	}
	pending, err := s.pendingQuestClaims(ctx, claims.UserID)
	if err != nil {
		return HealthSyncResult{}, err
	}
	dashboard, err := s.Dashboard(ctx, token)
	if err != nil {
		return HealthSyncResult{}, err
	}
	created := 0
	if inserted {
		created = 1
	}
	return HealthSyncResult{CreatedClaims: created, PendingClaims: pending, Dashboard: dashboard}, nil
}

func (s Service) ClaimQuest(ctx context.Context, token string, claimID string) (Dashboard, error) {
	claims, err := parseClaims(s.jwtSecret, token)
	if err != nil {
		return Dashboard{}, err
	}
	if s.integrations == nil {
		return Dashboard{}, ErrGoogleHealthNotConfigured
	}
	claim, err := s.integrations.GetQuestClaim(ctx, claims.UserID, claimID)
	if err != nil {
		return Dashboard{}, ErrQuestClaimNotFound
	}
	if claim.Status == QuestClaimStatusClaimed {
		return Dashboard{}, ErrQuestClaimAlreadyClaimed
	}
	if claim.Status != QuestClaimStatusPending {
		return Dashboard{}, ErrQuestClaimNotFound
	}
	allClaims, err := s.integrations.ListQuestClaims(ctx, claims.UserID)
	if err != nil {
		return Dashboard{}, err
	}
	if !questClaimIsClaimable(claim, allClaims) {
		return Dashboard{}, ErrQuestClaimLocked
	}
	_, activity, err := s.createActivityAndAward(ctx, token, map[string]any{
		"type":       claim.Type,
		"notes":      claim.Evidence,
		"occurredAt": claim.OccurredAt,
	})
	if err != nil {
		return Dashboard{}, err
	}
	if err := s.integrations.MarkQuestClaimClaimed(ctx, claims.UserID, claimID, activity.ID, time.Now()); err != nil {
		return Dashboard{}, err
	}
	return s.Dashboard(ctx, token)
}

type healthQuery struct {
	dataType string
	filter   string
}

func healthQueries(start time.Time, now time.Time) []healthQuery {
	after := start.UTC().Format(time.RFC3339)
	civilAfter := start.Format("2006-01-02T15:04:05")
	stepsStart := startOfToday(start)
	if stepsStart.Before(now.AddDate(0, 0, -90)) {
		stepsStart = start
	}
	stepsCivilAfter := stepsStart.Format("2006-01-02T15:04:05")
	return []healthQuery{
		{dataType: "exercise", filter: fmt.Sprintf(`exercise.interval.civil_start_time >= "%s"`, civilAfter)},
		{dataType: "steps", filter: fmt.Sprintf(`steps.interval.civil_start_time >= "%s"`, stepsCivilAfter)},
		{dataType: "sleep", filter: fmt.Sprintf(`sleep.interval.end_time >= "%s"`, after)},
		{dataType: "hydration-log", filter: fmt.Sprintf(`hydration_log.interval.civil_start_time >= "%s"`, civilAfter)},
		{dataType: "nutrition-log", filter: fmt.Sprintf(`nutrition_log.interval.civil_start_time >= "%s"`, civilAfter)},
		{dataType: "weight", filter: fmt.Sprintf(`weight.sample_time.physical_time >= "%s"`, after)},
		{dataType: "body-fat", filter: fmt.Sprintf(`body_fat.sample_time.physical_time >= "%s"`, after)},
	}
}

func healthCandidates(userID string, pointsByType map[string][]HealthDataPoint) []QuestClaim {
	var candidates []QuestClaim
	for _, point := range pointsByType["exercise"] {
		if point.Exercise == nil {
			continue
		}
		activityType := questTypeForExercise(point.Exercise.ExerciseType)
		if activityType == "" || exerciseDuration(*point.Exercise) < 10*time.Minute {
			continue
		}
		occurredAt := intervalEnd(point.Exercise.Interval)
		if occurredAt.IsZero() {
			occurredAt = intervalStart(point.Exercise.Interval)
		}
		if occurredAt.IsZero() {
			continue
		}
		candidates = append(candidates, newQuestClaim(userID, activityType, QuestClaimSourceGoogleHealth, point.ID(), occurredAt, exerciseEvidence(*point.Exercise)))
	}
	candidates = append(candidates, stepsCandidates(userID, pointsByType["steps"])...)
	for _, point := range pointsByType["sleep"] {
		if point.Sleep == nil {
			continue
		}
		minutes, _ := strconv.Atoi(point.Sleep.Summary.MinutesAsleep)
		if minutes < sleepGoalHours*60 {
			continue
		}
		occurredAt := intervalEnd(point.Sleep.Interval)
		if occurredAt.IsZero() {
			continue
		}
		candidates = append(candidates, newQuestClaim(userID, "sleep", QuestClaimSourceGoogleHealth, point.ID(), occurredAt, sleepEvidence(minutes)))
	}
	candidates = append(candidates, hydrationCandidates(userID, pointsByType["hydration-log"])...)
	for _, point := range pointsByType["nutrition-log"] {
		if point.NutritionLog == nil {
			continue
		}
		occurredAt := intervalEnd(point.NutritionLog.Interval)
		if occurredAt.IsZero() {
			occurredAt = intervalStart(point.NutritionLog.Interval)
		}
		if occurredAt.IsZero() {
			continue
		}
		candidates = append(candidates, newQuestClaim(userID, "healthy_meal", QuestClaimSourceGoogleHealth, point.ID(), occurredAt, "Nutrition logged"))
	}
	candidates = append(candidates, scaleCandidates(userID, pointsByType["weight"], pointsByType["body-fat"])...)
	return candidates
}

type dailyStepsAggregate struct {
	count      int
	occurredAt time.Time
}

func stepsCandidates(userID string, points []HealthDataPoint) []QuestClaim {
	byDate := map[string]dailyStepsAggregate{}
	for _, point := range points {
		if point.Steps == nil {
			continue
		}
		count, err := strconv.Atoi(point.Steps.Count)
		if err != nil || count <= 0 {
			continue
		}
		questDate := intervalCivilDateKey(point.Steps.Interval)
		occurredAt := civilDateOccurredAt(point.Steps.Interval.CivilStartTime)
		if questDate == "" {
			occurredAt = intervalStart(point.Steps.Interval)
			questDate = dateKey(occurredAt)
		}
		if occurredAt.IsZero() {
			occurredAt = intervalStart(point.Steps.Interval)
		}
		if occurredAt.IsZero() {
			continue
		}
		aggregate := byDate[questDate]
		aggregate.count += count
		if aggregate.occurredAt.IsZero() || occurredAt.After(aggregate.occurredAt) {
			aggregate.occurredAt = occurredAt
		}
		byDate[questDate] = aggregate
	}
	candidates := make([]QuestClaim, 0, len(byDate))
	for questDate, aggregate := range byDate {
		occurredAt := aggregate.occurredAt
		if dateKey(occurredAt) != questDate {
			parsed, err := time.ParseInLocation("2006-01-02", questDate, time.UTC)
			if err == nil {
				occurredAt = parsed.Add(12 * time.Hour)
			}
		}
		for _, rule := range stepTierRules() {
			if aggregate.count < rule.ThresholdValue {
				continue
			}
			candidates = append(candidates, newQuestClaim(
				userID,
				rule.Type,
				QuestClaimSourceGoogleHealth,
				fmt.Sprintf("google-health-steps-%s-%s", questDate, rule.Type),
				occurredAt,
				fmt.Sprintf("%d steps", aggregate.count),
			))
		}
	}
	sortQuestClaims(candidates)
	return candidates
}

type dailyHydrationAggregate struct {
	milliliters float64
	occurredAt  time.Time
}

func hydrationCandidates(userID string, points []HealthDataPoint) []QuestClaim {
	byDate := map[string]dailyHydrationAggregate{}
	for _, point := range points {
		if point.HydrationLog == nil || point.HydrationLog.AmountConsumed.Milliliters <= 0 {
			continue
		}
		questDate := intervalCivilDateKey(point.HydrationLog.Interval)
		occurredAt := civilDateOccurredAt(point.HydrationLog.Interval.CivilStartTime)
		if questDate == "" {
			occurredAt = intervalEnd(point.HydrationLog.Interval)
			if occurredAt.IsZero() {
				occurredAt = intervalStart(point.HydrationLog.Interval)
			}
			questDate = dateKey(occurredAt)
		}
		if occurredAt.IsZero() {
			occurredAt = intervalEnd(point.HydrationLog.Interval)
		}
		if occurredAt.IsZero() {
			occurredAt = intervalStart(point.HydrationLog.Interval)
		}
		if occurredAt.IsZero() {
			continue
		}
		aggregate := byDate[questDate]
		aggregate.milliliters += point.HydrationLog.AmountConsumed.Milliliters
		if aggregate.occurredAt.IsZero() || occurredAt.After(aggregate.occurredAt) {
			aggregate.occurredAt = occurredAt
		}
		byDate[questDate] = aggregate
	}
	candidates := make([]QuestClaim, 0, len(byDate))
	for questDate, aggregate := range byDate {
		occurredAt := aggregate.occurredAt
		if dateKey(occurredAt) != questDate {
			parsed, err := time.ParseInLocation("2006-01-02", questDate, time.UTC)
			if err == nil {
				occurredAt = parsed.Add(12 * time.Hour)
			}
		}
		total := int(aggregate.milliliters)
		for _, rule := range hydrationTierRules() {
			if total < rule.ThresholdValue {
				continue
			}
			candidates = append(candidates, newQuestClaim(
				userID,
				rule.Type,
				QuestClaimSourceGoogleHealth,
				fmt.Sprintf("google-health-hydration-%s-%s", questDate, rule.Type),
				occurredAt,
				fmt.Sprintf("%d ml hydration logged", total),
			))
		}
	}
	sortQuestClaims(candidates)
	return candidates
}

func scaleCandidates(userID string, weights []HealthDataPoint, bodyFats []HealthDataPoint) []QuestClaim {
	byDate := map[string]QuestClaim{}
	bodyFatByDate := map[string]float64{}
	for _, point := range bodyFats {
		if point.BodyFat == nil {
			continue
		}
		measuredAt := sampleTime(point.BodyFat.SampleTime)
		if measuredAt.IsZero() {
			continue
		}
		bodyFatByDate[dateKey(measuredAt)] = point.BodyFat.Percentage
		byDate[dateKey(measuredAt)] = newQuestClaim(userID, "scale_measurement", QuestClaimSourceGoogleHealth, point.ID(), measuredAt, fmt.Sprintf("Body fat %.1f%%", point.BodyFat.Percentage))
	}
	for _, point := range weights {
		if point.Weight == nil {
			continue
		}
		measuredAt := sampleTime(point.Weight.SampleTime)
		if measuredAt.IsZero() {
			continue
		}
		key := dateKey(measuredAt)
		evidence := fmt.Sprintf("Weight %.1f kg", point.Weight.WeightGrams/1000)
		if bodyFat, ok := bodyFatByDate[key]; ok {
			evidence = fmt.Sprintf("%s, body fat %.1f%%", evidence, bodyFat)
		}
		byDate[key] = newQuestClaim(userID, "scale_measurement", QuestClaimSourceGoogleHealth, point.ID(), measuredAt, evidence)
	}
	candidates := make([]QuestClaim, 0, len(byDate))
	for _, candidate := range byDate {
		candidates = append(candidates, candidate)
	}
	return candidates
}

func newQuestClaim(userID string, claimType string, source string, sourceID string, occurredAt time.Time, evidence string) QuestClaim {
	rule := ruleForType(claimType)
	return QuestClaim{
		ID:         uuid.NewString(),
		UserID:     userID,
		Type:       rule.Type,
		Title:      rule.Title,
		XP:         rule.XP,
		Stat:       rule.Stat,
		Source:     source,
		SourceID:   sourceID,
		Evidence:   evidence,
		OccurredAt: occurredAt,
		QuestDate:  dateKey(occurredAt),
		Status:     QuestClaimStatusPending,
	}
}

func ruleForType(ruleType string) ActivityRule {
	for _, rule := range localActivityRules() {
		if rule.Type == ruleType {
			return rule
		}
	}
	return ActivityRule{Type: ruleType, Title: ruleType, XP: 0, Stat: ""}
}

func localActivityRules() []ActivityRule {
	return applyActivityRuleCategoryColors([]ActivityRule{
		{Type: "cardio", Title: "Cardio Session", XP: 30, Stat: "cardio", Icon: "flame"},
		{Type: "daily_steps_bronze", Title: "Daily Steps — Bronze", XP: 20, Stat: "cardio", Icon: "footprints", Tier: "Bronze", ThresholdValue: 6000, ThresholdUnit: "steps", FollowUpType: "daily_steps_silver"},
		{Type: "daily_steps_silver", Title: "Daily Steps — Silver", XP: 30, Stat: "cardio", Icon: "footprints", Tier: "Silver", ThresholdValue: 8000, ThresholdUnit: "steps", FollowUpType: "daily_steps_gold", PrerequisiteType: "daily_steps_bronze"},
		{Type: "daily_steps_gold", Title: "Daily Steps — Gold", XP: 45, Stat: "cardio", Icon: "footprints", Tier: "Gold", ThresholdValue: 10000, ThresholdUnit: "steps", FollowUpType: "daily_steps_diamond", PrerequisiteType: "daily_steps_silver"},
		{Type: "daily_steps_diamond", Title: "Daily Steps — Diamond", XP: 70, Stat: "cardio", Icon: "footprints", Tier: "Diamond", ThresholdValue: 15000, ThresholdUnit: "steps", PrerequisiteType: "daily_steps_gold"},
		{Type: "exercise", Title: "Strength Session", XP: 40, Stat: "strength", Icon: "dumbbell"},
		{Type: "healthy_meal", Title: "Nourishing Meal", XP: 25, Stat: "fuel", Icon: "apple"},
		{Type: "hydration_bronze", Title: "Hydration Boost — Bronze", XP: 10, Stat: "fuel", Icon: "droplet", Tier: "Bronze", ThresholdValue: 500, ThresholdUnit: "ml", FollowUpType: "hydration_silver"},
		{Type: "hydration_silver", Title: "Hydration Boost — Silver", XP: 15, Stat: "fuel", Icon: "droplet", Tier: "Silver", ThresholdValue: 1000, ThresholdUnit: "ml", FollowUpType: "hydration_gold", PrerequisiteType: "hydration_bronze"},
		{Type: "hydration_gold", Title: "Hydration Boost — Gold", XP: 20, Stat: "fuel", Icon: "droplet", Tier: "Gold", ThresholdValue: 1500, ThresholdUnit: "ml", FollowUpType: "hydration_diamond", PrerequisiteType: "hydration_silver"},
		{Type: "hydration_diamond", Title: "Hydration Boost — Diamond", XP: 30, Stat: "fuel", Icon: "droplet", Tier: "Diamond", ThresholdValue: 2000, ThresholdUnit: "ml", PrerequisiteType: "hydration_gold"},
		{Type: "sleep", Title: "Sleep Goal Met", XP: 35, Stat: "recovery", Icon: "moon", ThresholdValue: sleepGoalHours, ThresholdUnit: "hours"},
		{Type: "mindfulness", Title: "Mindset Moment", XP: 20, Stat: "mindset", Icon: "sparkles"},
		{Type: "recovery", Title: "Recovery Ritual", XP: 20, Stat: "recovery", Icon: "heart-pulse"},
		{Type: "scale_measurement", Title: "Scale Measurement", XP: 15, Stat: "biometrics", Icon: "scale"},
		{Type: "waist_to_height_ratio", Title: "Waist-to-Height Ratio", XP: 15, Stat: "biometrics", Icon: "ruler"},
	})
}

func stepTierRules() []ActivityRule {
	return tierRules("daily_steps_")
}

func hydrationTierRules() []ActivityRule {
	return tierRules("hydration_")
}

func tierRules(prefix string) []ActivityRule {
	rules := []ActivityRule{}
	for _, rule := range localActivityRules() {
		if strings.HasPrefix(rule.Type, prefix) {
			rules = append(rules, rule)
		}
	}
	return rules
}

func claimableQuestClaims(claims []QuestClaim) []QuestClaim {
	claimable := make([]QuestClaim, 0, len(claims))
	for _, claim := range claims {
		if claim.Status != QuestClaimStatusPending {
			continue
		}
		if questClaimIsClaimable(claim, claims) {
			claimable = append(claimable, claimWithRule(claim))
		}
	}
	sortQuestClaims(claimable)
	return claimable
}

func questClaimIsClaimable(claim QuestClaim, claims []QuestClaim) bool {
	rule := ruleForType(claim.Type)
	if rule.PrerequisiteType == "" {
		return true
	}
	for _, existing := range claims {
		if existing.UserID == claim.UserID &&
			existing.Type == rule.PrerequisiteType &&
			existing.QuestDate == claim.QuestDate &&
			existing.Status == QuestClaimStatusClaimed {
			return true
		}
	}
	return false
}

func claimWithRule(claim QuestClaim) QuestClaim {
	rule := ruleForType(claim.Type)
	claim.Title = rule.Title
	claim.XP = rule.XP
	claim.Stat = rule.Stat
	return claim
}

func sortQuestClaims(claims []QuestClaim) {
	ranks := map[string]int{}
	for index, rule := range localActivityRules() {
		ranks[rule.Type] = index
	}
	sort.SliceStable(claims, func(i int, j int) bool {
		if claims[i].QuestDate != claims[j].QuestDate {
			return claims[i].QuestDate < claims[j].QuestDate
		}
		if !claims[i].OccurredAt.Equal(claims[j].OccurredAt) {
			return claims[i].OccurredAt.Before(claims[j].OccurredAt)
		}
		leftRank, ok := ranks[claims[i].Type]
		if !ok {
			leftRank = len(ranks)
		}
		rightRank, ok := ranks[claims[j].Type]
		if !ok {
			rightRank = len(ranks)
		}
		return leftRank < rightRank
	})
}

func questTypeForExercise(exerciseType string) string {
	value := strings.ToUpper(exerciseType)
	switch {
	case strings.Contains(value, "RUN") || strings.Contains(value, "BIKE") || strings.Contains(value, "CYCL") || strings.Contains(value, "SWIM") || strings.Contains(value, "ROW") || strings.Contains(value, "ELLIPTICAL") || strings.Contains(value, "CARDIO"):
		return "cardio"
	case strings.Contains(value, "STRENGTH") || strings.Contains(value, "WEIGHT") || strings.Contains(value, "RESISTANCE") || strings.Contains(value, "CIRCUIT"):
		return "exercise"
	case strings.Contains(value, "YOGA") || strings.Contains(value, "PILATES") || strings.Contains(value, "STRETCH") || strings.Contains(value, "TAI_CHI") || strings.Contains(value, "MOBILITY"):
		return "recovery"
	default:
		return ""
	}
}

func exerciseDuration(exercise HealthExercise) time.Duration {
	if exercise.ActiveDuration != "" {
		if duration, err := time.ParseDuration(exercise.ActiveDuration); err == nil {
			return duration
		}
	}
	start := intervalStart(exercise.Interval)
	end := intervalEnd(exercise.Interval)
	if !start.IsZero() && !end.IsZero() && end.After(start) {
		return end.Sub(start)
	}
	return 0
}

func exerciseEvidence(exercise HealthExercise) string {
	duration := exerciseDuration(exercise)
	parts := []string{displayExerciseType(exercise.ExerciseType)}
	if duration > 0 {
		parts = append(parts, fmt.Sprintf("%d min", int(duration.Minutes())))
	}
	if exercise.MetricsSummary.DistanceMillimeters > 0 {
		parts = append(parts, fmt.Sprintf("%.1f km", exercise.MetricsSummary.DistanceMillimeters/1000000))
	}
	return strings.Join(parts, " · ")
}

func sleepEvidence(minutes int) string {
	hours := minutes / 60
	remainder := minutes % 60
	return fmt.Sprintf("%s %s asleep", pluralize(hours, "hour"), pluralize(remainder, "minute"))
}

func pluralize(value int, unit string) string {
	if value == 1 {
		return fmt.Sprintf("%d %s", value, unit)
	}
	return fmt.Sprintf("%d %ss", value, unit)
}

func displayExerciseType(value string) string {
	value = strings.ReplaceAll(strings.ToLower(value), "_", " ")
	if value == "" {
		return "Exercise"
	}
	return strings.Title(value)
}

func (point HealthDataPoint) ID() string {
	if point.DataPointName != "" {
		return point.DataPointName
	}
	if point.Name != "" {
		return point.Name
	}
	return uuid.NewString()
}

func intervalStart(interval HealthInterval) time.Time {
	return parseHealthTime(interval.StartTime)
}

func intervalEnd(interval HealthInterval) time.Time {
	return parseHealthTime(interval.EndTime)
}

func sampleTime(sample HealthSampleTime) time.Time {
	return parseHealthTime(sample.PhysicalTime)
}

func intervalCivilDateKey(interval HealthInterval) string {
	return civilDateKey(interval.CivilStartTime)
}

func civilDateKey(value HealthCivilDateTime) string {
	if value.Date.Year == 0 || value.Date.Month == 0 || value.Date.Day == 0 {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", value.Date.Year, value.Date.Month, value.Date.Day)
}

func civilDateOccurredAt(value HealthCivilDateTime) time.Time {
	if value.Date.Year == 0 || value.Date.Month == 0 || value.Date.Day == 0 {
		return time.Time{}
	}
	return time.Date(value.Date.Year, time.Month(value.Date.Month), value.Date.Day, 12, 0, 0, 0, time.UTC)
}

func parseHealthTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func dateKey(value time.Time) string {
	return value.Format("2006-01-02")
}

func startOfToday(now time.Time) time.Time {
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, now.Location())
}
