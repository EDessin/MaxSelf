package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/EDessin/MaxSelf/internal/identity/domain"
	"gorm.io/gorm"
)

type UserModel struct {
	ID           string `gorm:"primaryKey;type:uuid"`
	Email        string `gorm:"uniqueIndex;not null"`
	DisplayName  string `gorm:"not null"`
	AvatarURL    string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AuthIdentityModel struct {
	ID             string `gorm:"primaryKey;type:uuid"`
	UserID         string `gorm:"type:uuid;index;not null"`
	Provider       string `gorm:"uniqueIndex:idx_provider_user;not null"`
	ProviderUserID string `gorm:"uniqueIndex:idx_provider_user;not null"`
	Email          string `gorm:"not null"`
	CreatedAt      time.Time
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&UserModel{}, &AuthIdentityModel{})
}

func (r Repository) CreateUser(ctx context.Context, user domain.User, identity domain.AuthIdentity) (domain.User, error) {
	userModel := fromUser(user)
	identityModel := fromIdentity(identity)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&userModel).Error; err != nil {
			return err
		}
		return tx.Create(&identityModel).Error
	})
	return toUser(userModel), err
}

func (r Repository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	var model UserModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		return domain.User{}, err
	}
	return toUser(model), nil
}

func (r Repository) FindByProvider(ctx context.Context, provider domain.Provider, providerUserID string) (domain.User, error) {
	var identity AuthIdentityModel
	if err := r.db.WithContext(ctx).Where("provider = ? AND provider_user_id = ?", string(provider), providerUserID).First(&identity).Error; err != nil {
		return domain.User{}, err
	}
	var user UserModel
	if err := r.db.WithContext(ctx).Where("id = ?", identity.UserID).First(&user).Error; err != nil {
		return domain.User{}, err
	}
	return toUser(user), nil
}

func (r Repository) LinkIdentity(ctx context.Context, identity domain.AuthIdentity) error {
	model := fromIdentity(identity)
	err := r.db.WithContext(ctx).Create(&model).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil
	}
	return err
}

func fromUser(user domain.User) UserModel {
	return UserModel{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AvatarURL:    user.AvatarURL,
		PasswordHash: user.PasswordHash,
	}
}

func toUser(model UserModel) domain.User {
	return domain.User{
		ID:           model.ID,
		Email:        model.Email,
		DisplayName:  model.DisplayName,
		AvatarURL:    model.AvatarURL,
		PasswordHash: model.PasswordHash,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func fromIdentity(identity domain.AuthIdentity) AuthIdentityModel {
	return AuthIdentityModel{
		ID:             identity.ID,
		UserID:         identity.UserID,
		Provider:       string(identity.Provider),
		ProviderUserID: identity.ProviderUserID,
		Email:          identity.Email,
	}
}
