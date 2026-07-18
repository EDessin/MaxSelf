package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/EDessin/MaxSelf/internal/facade/application"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HealthAuthStateModel struct {
	State     string `gorm:"primaryKey"`
	UserID    string `gorm:"type:uuid;index;not null"`
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type HealthConnectionModel struct {
	UserID       string `gorm:"primaryKey;type:uuid"`
	AccessToken  string `gorm:"type:text"`
	RefreshToken string `gorm:"type:text"`
	TokenType    string
	Scope        string `gorm:"type:text"`
	Expiry       time.Time
	LastSyncedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type QuestClaimModel struct {
	ID         string `gorm:"primaryKey;type:uuid"`
	UserID     string `gorm:"type:uuid;uniqueIndex:idx_user_claim_day;index;not null"`
	Type       string `gorm:"uniqueIndex:idx_user_claim_day;not null"`
	Title      string `gorm:"not null"`
	XP         int    `gorm:"not null"`
	Stat       string `gorm:"not null"`
	Source     string `gorm:"not null"`
	SourceID   string
	Evidence   string `gorm:"type:text"`
	OccurredAt time.Time
	QuestDate  string `gorm:"uniqueIndex:idx_user_claim_day;not null"`
	Status     string `gorm:"index;not null"`
	ActivityID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ClaimedAt  *time.Time
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&HealthAuthStateModel{}, &HealthConnectionModel{}, &QuestClaimModel{})
}

func (r Repository) SaveHealthAuthState(ctx context.Context, state application.HealthAuthState) error {
	return r.db.WithContext(ctx).Create(&HealthAuthStateModel{
		State:     state.State,
		UserID:    state.UserID,
		ExpiresAt: state.ExpiresAt,
		UsedAt:    state.UsedAt,
	}).Error
}

func (r Repository) ConsumeHealthAuthState(ctx context.Context, state string, now time.Time) (application.HealthAuthState, error) {
	var model HealthAuthStateModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("state = ?", state).First(&model).Error; err != nil {
			return err
		}
		if model.UsedAt != nil || !model.ExpiresAt.After(now) {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&model).Update("used_at", now).Error
	})
	if err != nil {
		return application.HealthAuthState{}, err
	}
	usedAt := now
	return application.HealthAuthState{
		State:     model.State,
		UserID:    model.UserID,
		ExpiresAt: model.ExpiresAt,
		UsedAt:    &usedAt,
		CreatedAt: model.CreatedAt,
	}, nil
}

func (r Repository) SaveHealthConnection(ctx context.Context, connection application.HealthConnection) error {
	model := HealthConnectionModel{
		UserID:       connection.UserID,
		AccessToken:  connection.AccessToken,
		RefreshToken: connection.RefreshToken,
		TokenType:    connection.TokenType,
		Scope:        connection.Scope,
		Expiry:       connection.Expiry,
		LastSyncedAt: connection.LastSyncedAt,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"access_token",
			"refresh_token",
			"token_type",
			"scope",
			"expiry",
			"last_synced_at",
			"updated_at",
		}),
	}).Create(&model).Error
}

func (r Repository) GetHealthConnection(ctx context.Context, userID string) (application.HealthConnection, error) {
	var model HealthConnectionModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error; err != nil {
		return application.HealthConnection{}, err
	}
	return toHealthConnection(model), nil
}

func (r Repository) UpdateHealthConnectionSync(ctx context.Context, userID string, syncedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&HealthConnectionModel{}).Where("user_id = ?", userID).Update("last_synced_at", syncedAt).Error
}

func (r Repository) UpsertQuestClaim(ctx context.Context, claim application.QuestClaim) (application.QuestClaim, bool, error) {
	model := fromQuestClaim(claim)
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "type"}, {Name: "quest_date"}},
		DoNothing: true,
	}).Create(&model)
	if result.Error != nil {
		return application.QuestClaim{}, false, result.Error
	}
	if result.RowsAffected > 0 {
		return toQuestClaim(model), true, nil
	}
	var existing QuestClaimModel
	if err := r.db.WithContext(ctx).Where("user_id = ? AND type = ? AND quest_date = ?", claim.UserID, claim.Type, claim.QuestDate).First(&existing).Error; err != nil {
		return application.QuestClaim{}, false, err
	}
	return toQuestClaim(existing), false, nil
}

func (r Repository) ListQuestClaims(ctx context.Context, userID string) ([]application.QuestClaim, error) {
	var models []QuestClaimModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("quest_date ASC, occurred_at ASC, created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	claims := make([]application.QuestClaim, 0, len(models))
	for _, model := range models {
		claims = append(claims, toQuestClaim(model))
	}
	return claims, nil
}

func (r Repository) ListPendingQuestClaims(ctx context.Context, userID string) ([]application.QuestClaim, error) {
	var models []QuestClaimModel
	if err := r.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, application.QuestClaimStatusPending).Order("occurred_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	claims := make([]application.QuestClaim, 0, len(models))
	for _, model := range models {
		claims = append(claims, toQuestClaim(model))
	}
	return claims, nil
}

func (r Repository) CountPendingQuestClaims(ctx context.Context, userID string) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&QuestClaimModel{}).Where("user_id = ? AND status = ?", userID, application.QuestClaimStatusPending).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r Repository) GetQuestClaim(ctx context.Context, userID string, claimID string) (application.QuestClaim, error) {
	var model QuestClaimModel
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", claimID, userID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return application.QuestClaim{}, application.ErrQuestClaimNotFound
		}
		return application.QuestClaim{}, err
	}
	return toQuestClaim(model), nil
}

func (r Repository) MarkQuestClaimClaimed(ctx context.Context, userID string, claimID string, activityID string, claimedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&QuestClaimModel{}).
		Where("id = ? AND user_id = ? AND status = ?", claimID, userID, application.QuestClaimStatusPending).
		Updates(map[string]any{
			"status":      application.QuestClaimStatusClaimed,
			"activity_id": activityID,
			"claimed_at":  claimedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return application.ErrQuestClaimAlreadyClaimed
	}
	return nil
}

func toHealthConnection(model HealthConnectionModel) application.HealthConnection {
	return application.HealthConnection{
		UserID:       model.UserID,
		AccessToken:  model.AccessToken,
		RefreshToken: model.RefreshToken,
		TokenType:    model.TokenType,
		Scope:        model.Scope,
		Expiry:       model.Expiry,
		LastSyncedAt: model.LastSyncedAt,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func fromQuestClaim(claim application.QuestClaim) QuestClaimModel {
	return QuestClaimModel{
		ID:         claim.ID,
		UserID:     claim.UserID,
		Type:       claim.Type,
		Title:      claim.Title,
		XP:         claim.XP,
		Stat:       claim.Stat,
		Source:     claim.Source,
		SourceID:   claim.SourceID,
		Evidence:   claim.Evidence,
		OccurredAt: claim.OccurredAt,
		QuestDate:  claim.QuestDate,
		Status:     claim.Status,
		ActivityID: claim.ActivityID,
		CreatedAt:  claim.CreatedAt,
		ClaimedAt:  claim.ClaimedAt,
	}
}

func toQuestClaim(model QuestClaimModel) application.QuestClaim {
	return application.QuestClaim{
		ID:         model.ID,
		UserID:     model.UserID,
		Type:       model.Type,
		Title:      model.Title,
		XP:         model.XP,
		Stat:       model.Stat,
		Source:     model.Source,
		SourceID:   model.SourceID,
		Evidence:   model.Evidence,
		OccurredAt: model.OccurredAt,
		QuestDate:  model.QuestDate,
		Status:     model.Status,
		ActivityID: model.ActivityID,
		CreatedAt:  model.CreatedAt,
		ClaimedAt:  model.ClaimedAt,
	}
}
