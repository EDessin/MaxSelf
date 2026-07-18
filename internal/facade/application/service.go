package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

type Service struct {
	identity     Client
	activity     Client
	progress     Client
	jwtSecret    string
	integrations IntegrationRepository
	googleHealth GoogleHealthClient
}

type AuthResult struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
}

type ActivityRule struct {
	Type             string `json:"type"`
	Title            string `json:"title"`
	XP               int    `json:"xp"`
	Stat             string `json:"stat"`
	Icon             string `json:"icon"`
	Color            string `json:"color"`
	Tier             string `json:"tier,omitempty"`
	ThresholdValue   int    `json:"thresholdValue,omitempty"`
	ThresholdUnit    string `json:"thresholdUnit,omitempty"`
	FollowUpType     string `json:"followUpType,omitempty"`
	PrerequisiteType string `json:"prerequisiteType,omitempty"`
}

type Activity struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Notes      string    `json:"notes"`
	XP         int       `json:"xp"`
	Stat       string    `json:"stat"`
	OccurredAt time.Time `json:"occurredAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Progress struct {
	UserID           string         `json:"userId"`
	Level            int            `json:"level"`
	TotalXP          int            `json:"totalXp"`
	CurrentLevelXP   int            `json:"currentLevelXp"`
	NextLevelXP      int            `json:"nextLevelXp"`
	StreakDays       int            `json:"streakDays"`
	LastActivityDate *time.Time     `json:"lastActivityDate"`
	Stats            map[string]int `json:"stats"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type Dashboard struct {
	User         User                    `json:"user"`
	Progress     Progress                `json:"progress"`
	Activities   []Activity              `json:"activities"`
	Rules        []ActivityRule          `json:"rules"`
	GoogleHealth HealthIntegrationStatus `json:"googleHealth"`
	QuestClaims  []QuestClaim            `json:"questClaims"`
}

func NewService(identity, activity, progress Client, jwtSecret string) Service {
	return Service{identity: identity, activity: activity, progress: progress, jwtSecret: jwtSecret}
}

func NewServiceWithIntegrations(identity, activity, progress Client, jwtSecret string, integrations IntegrationRepository, googleHealth GoogleHealthClient) Service {
	return Service{identity: identity, activity: activity, progress: progress, jwtSecret: jwtSecret, integrations: integrations, googleHealth: googleHealth}
}

func (s Service) Register(ctx context.Context, req any) (AuthResult, error) {
	var result AuthResult
	err := s.identity.Post(ctx, "/auth/register", nil, req, &result)
	return result, err
}

func (s Service) Login(ctx context.Context, req any) (AuthResult, error) {
	var result AuthResult
	err := s.identity.Post(ctx, "/auth/login", nil, req, &result)
	return result, err
}

func (s Service) Me(ctx context.Context, token string) (User, error) {
	var user User
	err := s.identity.Get(ctx, "/users/me", map[string]string{"Authorization": "Bearer " + token}, &user)
	return user, err
}

func (s Service) Dashboard(ctx context.Context, token string) (Dashboard, error) {
	claims, err := parseClaims(s.jwtSecret, token)
	if err != nil {
		return Dashboard{}, err
	}
	user, err := s.Me(ctx, token)
	if err != nil {
		return Dashboard{}, err
	}
	progress, err := s.Progress(ctx, claims.UserID)
	if err != nil {
		return Dashboard{}, err
	}
	activities, err := s.Activities(ctx, claims.UserID)
	if err != nil {
		return Dashboard{}, err
	}
	rules, err := s.ActivityRules(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		User:         user,
		Progress:     progress,
		Activities:   activities,
		Rules:        rules,
		GoogleHealth: s.GoogleHealthStatus(ctx, claims.UserID),
		QuestClaims:  s.PendingQuestClaims(ctx, claims.UserID),
	}, nil
}

func (s Service) CreateActivity(ctx context.Context, token string, req any) (Dashboard, error) {
	return Dashboard{}, errors.New("manual XP claims are disabled; sync health data and claim a quest instead")
}

func (s Service) createActivityAndAward(ctx context.Context, token string, req any) (Dashboard, Activity, error) {
	claims, err := parseClaims(s.jwtSecret, token)
	if err != nil {
		return Dashboard{}, Activity{}, err
	}
	user, err := s.Me(ctx, token)
	if err != nil {
		return Dashboard{}, Activity{}, err
	}
	var activity Activity
	if err := s.activity.Post(ctx, "/activities", map[string]string{"X-User-ID": claims.UserID}, req, &activity); err != nil {
		return Dashboard{}, Activity{}, err
	}
	award := map[string]any{
		"userId":     claims.UserID,
		"activityId": activity.ID,
		"xp":         activity.XP,
		"stat":       activity.Stat,
		"occurredAt": activity.OccurredAt,
	}
	var progress Progress
	if err := s.progress.Post(ctx, "/progress/award", nil, award, &progress); err != nil {
		return Dashboard{}, Activity{}, err
	}
	activities, err := s.Activities(ctx, claims.UserID)
	if err != nil {
		return Dashboard{}, Activity{}, err
	}
	rules, err := s.ActivityRules(ctx)
	if err != nil {
		return Dashboard{}, Activity{}, err
	}
	return Dashboard{
		User:         user,
		Progress:     progress,
		Activities:   activities,
		Rules:        rules,
		GoogleHealth: s.GoogleHealthStatus(ctx, claims.UserID),
		QuestClaims:  s.PendingQuestClaims(ctx, claims.UserID),
	}, activity, nil
}

func (s Service) Activities(ctx context.Context, userID string) ([]Activity, error) {
	var activities []Activity
	err := s.activity.Get(ctx, "/activities?limit=20", map[string]string{"X-User-ID": userID}, &activities)
	return activities, err
}

func (s Service) ActivityRules(ctx context.Context) ([]ActivityRule, error) {
	var rules []ActivityRule
	err := s.activity.Get(ctx, "/activity-types", nil, &rules)
	if err != nil {
		return nil, err
	}
	return applyActivityRuleCategoryColors(rules), nil
}

func activityRuleCategoryColor(stat string) string {
	switch stat {
	case "cardio":
		return "#f59e0b"
	case "strength":
		return "#ff5a5f"
	case "fuel":
		return "#22c55e"
	case "recovery":
		return "#6366f1"
	case "mindset":
		return "#a855f7"
	case "biometrics":
		return "#0891b2"
	default:
		return "#f59e0b"
	}
}

func applyActivityRuleCategoryColors(rules []ActivityRule) []ActivityRule {
	for index := range rules {
		rules[index].Color = activityRuleCategoryColor(rules[index].Stat)
	}
	return rules
}

func (s Service) Progress(ctx context.Context, userID string) (Progress, error) {
	var progress Progress
	err := s.progress.Get(ctx, fmt.Sprintf("/progress/%s", userID), nil, &progress)
	return progress, err
}

func parseClaims(jwtSecret string, token string) (httpx.Claims, error) {
	return httpx.ParseToken(jwtSecret, token)
}
