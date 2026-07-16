package application

import (
	"context"
	"time"

	"github.com/EDessin/MaxSelf/internal/activity/domain"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, activity domain.Activity) (domain.Activity, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]domain.Activity, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return Service{repository: repository}
}

func (s Service) Create(ctx context.Context, userID string, activityType domain.ActivityType, notes string, occurredAt time.Time) (domain.Activity, error) {
	rule, err := domain.RuleFor(activityType)
	if err != nil {
		return domain.Activity{}, err
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	activity := domain.Activity{
		ID:         uuid.NewString(),
		UserID:     userID,
		Type:       activityType,
		Title:      rule.Title,
		Notes:      notes,
		XP:         rule.XP,
		Stat:       rule.Stat,
		OccurredAt: occurredAt,
	}
	return s.repository.Create(ctx, activity)
}

func (s Service) List(ctx context.Context, userID string, limit int) ([]domain.Activity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repository.ListByUser(ctx, userID, limit)
}

func (s Service) Rules() []domain.ActivityRule {
	return domain.Rules()
}
