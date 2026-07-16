package application

import (
	"context"
	"fmt"
	"time"

	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

type Service struct {
	identity  Client
	activity  Client
	progress  Client
	jwtSecret string
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
	Type  string `json:"type"`
	Title string `json:"title"`
	XP    int    `json:"xp"`
	Stat  string `json:"stat"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
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
	User       User           `json:"user"`
	Progress   Progress       `json:"progress"`
	Activities []Activity     `json:"activities"`
	Rules      []ActivityRule `json:"rules"`
}

func NewService(identity, activity, progress Client, jwtSecret string) Service {
	return Service{identity: identity, activity: activity, progress: progress, jwtSecret: jwtSecret}
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
	claims, err := httpx.ParseToken(s.jwtSecret, token)
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
	return Dashboard{User: user, Progress: progress, Activities: activities, Rules: rules}, nil
}

func (s Service) CreateActivity(ctx context.Context, token string, req any) (Dashboard, error) {
	claims, err := httpx.ParseToken(s.jwtSecret, token)
	if err != nil {
		return Dashboard{}, err
	}
	user, err := s.Me(ctx, token)
	if err != nil {
		return Dashboard{}, err
	}
	var activity Activity
	if err := s.activity.Post(ctx, "/activities", map[string]string{"X-User-ID": claims.UserID}, req, &activity); err != nil {
		return Dashboard{}, err
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
	return Dashboard{User: user, Progress: progress, Activities: activities, Rules: rules}, nil
}

func (s Service) Activities(ctx context.Context, userID string) ([]Activity, error) {
	var activities []Activity
	err := s.activity.Get(ctx, "/activities?limit=20", map[string]string{"X-User-ID": userID}, &activities)
	return activities, err
}

func (s Service) ActivityRules(ctx context.Context) ([]ActivityRule, error) {
	var rules []ActivityRule
	err := s.activity.Get(ctx, "/activity-types", nil, &rules)
	return rules, err
}

func (s Service) Progress(ctx context.Context, userID string) (Progress, error) {
	var progress Progress
	err := s.progress.Get(ctx, fmt.Sprintf("/progress/%s", userID), nil, &progress)
	return progress, err
}
