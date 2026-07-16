package postgres

import (
	"context"
	"time"

	"github.com/EDessin/MaxSelf/internal/progress/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserProgressModel struct {
	UserID           string `gorm:"primaryKey;type:uuid"`
	Level            int
	TotalXP          int
	CurrentLevelXP   int
	NextLevelXP      int
	StreakDays       int
	LastActivityDate *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type StatProgressModel struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    string `gorm:"type:uuid;uniqueIndex:idx_user_stat;not null"`
	Stat      string `gorm:"uniqueIndex:idx_user_stat;not null"`
	XP        int
	UpdatedAt time.Time
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&UserProgressModel{}, &StatProgressModel{})
}

func (r Repository) GetProfile(ctx context.Context, userID string) (domain.Profile, error) {
	var progress UserProgressModel
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&progress)
	if result.Error != nil {
		return domain.Profile{}, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.Profile{}, gorm.ErrRecordNotFound
	}
	var statModels []StatProgressModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&statModels).Error; err != nil {
		return domain.Profile{}, err
	}
	stats := map[domain.Stat]int{
		domain.StatCardio:              0,
		domain.StatStrength:            0,
		domain.StatFuel:                0,
		domain.StatRecovery:            0,
		domain.StatMindset:             0,
		domain.StatCardioConsistency:   0,
		domain.StatStrengthConsistency: 0,
		domain.StatFuelConsistency:     0,
		domain.StatRecoveryConsistency: 0,
		domain.StatMindsetConsistency:  0,
	}
	for _, model := range statModels {
		stats[domain.Stat(model.Stat)] = model.XP
	}
	return domain.Profile{
		UserID:           progress.UserID,
		Level:            progress.Level,
		TotalXP:          progress.TotalXP,
		CurrentLevelXP:   progress.CurrentLevelXP,
		NextLevelXP:      progress.NextLevelXP,
		StreakDays:       progress.StreakDays,
		LastActivityDate: progress.LastActivityDate,
		Stats:            stats,
		UpdatedAt:        progress.UpdatedAt,
	}, nil
}

func (r Repository) SaveProfile(ctx context.Context, profile domain.Profile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		progress := UserProgressModel{
			UserID:           profile.UserID,
			Level:            profile.Level,
			TotalXP:          profile.TotalXP,
			CurrentLevelXP:   profile.CurrentLevelXP,
			NextLevelXP:      profile.NextLevelXP,
			StreakDays:       profile.StreakDays,
			LastActivityDate: profile.LastActivityDate,
		}
		if err := tx.Save(&progress).Error; err != nil {
			return err
		}
		for stat, xp := range profile.Stats {
			model := StatProgressModel{UserID: profile.UserID, Stat: string(stat), XP: xp}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "stat"},
				},
				DoUpdates: clause.AssignmentColumns([]string{"xp", "updated_at"}),
			}).Create(&model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
