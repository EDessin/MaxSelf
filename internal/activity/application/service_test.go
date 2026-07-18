package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/activity/domain"
)

type fakeActivityRepository struct {
	created     domain.Activity
	createCalls int
	createErr   error
	listLimit   int
	listResult  []domain.Activity
	listErr     error
}

func (r *fakeActivityRepository) Create(_ context.Context, activity domain.Activity) (domain.Activity, error) {
	r.createCalls++
	r.created = activity
	if r.createErr != nil {
		return domain.Activity{}, r.createErr
	}
	return activity, nil
}

func (r *fakeActivityRepository) ListByUser(_ context.Context, _ string, limit int) ([]domain.Activity, error) {
	r.listLimit = limit
	return r.listResult, r.listErr
}

func TestCreateBuildsActivityFromRule(t *testing.T) {
	repo := &fakeActivityRepository{}
	service := NewService(repo)
	occurredAt := time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC)

	activity, err := service.Create(context.Background(), "user-1", domain.TypeSleep, "slept well", occurredAt)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if repo.createCalls != 1 {
		t.Fatalf("expected repository Create once, got %d", repo.createCalls)
	}
	if activity.ID == "" || activity.UserID != "user-1" || activity.Type != domain.TypeSleep {
		t.Fatalf("unexpected activity identity: %+v", activity)
	}
	if activity.Title != "Sleep Goal Met" || activity.XP != 35 || activity.Stat != domain.StatRecovery {
		t.Fatalf("activity did not use rule fields: %+v", activity)
	}
	if !activity.OccurredAt.Equal(occurredAt) || activity.Notes != "slept well" {
		t.Fatalf("activity did not preserve request fields: %+v", activity)
	}
}

func TestCreateDefaultsOccurredAtAndRejectsUnknownType(t *testing.T) {
	repo := &fakeActivityRepository{}
	service := NewService(repo)

	activity, err := service.Create(context.Background(), "user-1", domain.TypeHydrationBronze, "", time.Time{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if activity.OccurredAt.IsZero() {
		t.Fatal("expected occurredAt to default to now")
	}

	_, err = service.Create(context.Background(), "user-1", domain.ActivityType("nope"), "", time.Now())
	if !errors.Is(err, domain.ErrUnknownActivityType) {
		t.Fatalf("expected ErrUnknownActivityType, got %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("unknown type should not call repository, got %d calls", repo.createCalls)
	}
}

func TestListNormalizesLimitAndPropagatesErrors(t *testing.T) {
	expectedErr := errors.New("store failed")
	repo := &fakeActivityRepository{listErr: expectedErr}
	service := NewService(repo)

	_, err := service.List(context.Background(), "user-1", 0)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected list error, got %v", err)
	}
	if repo.listLimit != 20 {
		t.Fatalf("expected default limit 20, got %d", repo.listLimit)
	}

	repo.listErr = nil
	_, err = service.List(context.Background(), "user-1", 150)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if repo.listLimit != 20 {
		t.Fatalf("expected capped default limit 20, got %d", repo.listLimit)
	}

	_, err = service.List(context.Background(), "user-1", 12)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if repo.listLimit != 12 {
		t.Fatalf("expected requested limit 12, got %d", repo.listLimit)
	}
}

func TestRulesReturnsConfiguredRules(t *testing.T) {
	service := NewService(&fakeActivityRepository{})
	if got := service.Rules(); len(got) != len(domain.Rules()) {
		t.Fatalf("expected rules passthrough, got %d", len(got))
	}
}
