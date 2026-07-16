package postgres

import (
	"context"
	"time"

	"github.com/EDessin/MaxSelf/internal/activity/domain"
	"gorm.io/gorm"
)

type ActivityModel struct {
	ID         string `gorm:"primaryKey;type:uuid"`
	UserID     string `gorm:"type:uuid;index;not null"`
	Type       string `gorm:"not null"`
	Title      string `gorm:"not null"`
	Notes      string
	XP         int    `gorm:"not null"`
	Stat       string `gorm:"not null"`
	OccurredAt time.Time
	CreatedAt  time.Time
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&ActivityModel{})
}

func (r Repository) Create(ctx context.Context, activity domain.Activity) (domain.Activity, error) {
	model := fromActivity(activity)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.Activity{}, err
	}
	return toActivity(model), nil
}

func (r Repository) ListByUser(ctx context.Context, userID string, limit int) ([]domain.Activity, error) {
	var models []ActivityModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("occurred_at DESC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	activities := make([]domain.Activity, 0, len(models))
	for _, model := range models {
		activities = append(activities, toActivity(model))
	}
	return activities, nil
}

func fromActivity(activity domain.Activity) ActivityModel {
	return ActivityModel{
		ID:         activity.ID,
		UserID:     activity.UserID,
		Type:       string(activity.Type),
		Title:      activity.Title,
		Notes:      activity.Notes,
		XP:         activity.XP,
		Stat:       string(activity.Stat),
		OccurredAt: activity.OccurredAt,
	}
}

func toActivity(model ActivityModel) domain.Activity {
	return domain.Activity{
		ID:         model.ID,
		UserID:     model.UserID,
		Type:       domain.ActivityType(model.Type),
		Title:      model.Title,
		Notes:      model.Notes,
		XP:         model.XP,
		Stat:       domain.Stat(model.Stat),
		OccurredAt: model.OccurredAt,
		CreatedAt:  model.CreatedAt,
	}
}
