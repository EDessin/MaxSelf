package application

import (
	"context"

	"github.com/EDessin/MaxSelf/internal/progress/domain"
)

type Repository interface {
	GetProfile(ctx context.Context, userID string) (domain.Profile, error)
	SaveProfile(ctx context.Context, profile domain.Profile) error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return Service{repository: repository}
}

func (s Service) Award(ctx context.Context, award domain.Award) (domain.Profile, error) {
	profile, err := s.repository.GetProfile(ctx, award.UserID)
	if err != nil {
		profile = domain.Profile{
			UserID:      award.UserID,
			Level:       1,
			Stats:       map[domain.Stat]int{},
			NextLevelXP: domain.XPNeededForLevel(1),
		}
	}

	profile.TotalXP += award.XP
	profile.Stats[award.Stat] += award.XP
	if consistencyStat := domain.ConsistencyStatFor(award.Stat); consistencyStat != "" {
		profile.Stats[consistencyStat] += 5
	}
	profile.StreakDays, profile.LastActivityDate = domain.UpdatedStreak(profile.StreakDays, profile.LastActivityDate, award.OccurredAt)
	profile.Level, profile.CurrentLevelXP, profile.NextLevelXP = domain.LevelFor(profile.TotalXP)

	if err := s.repository.SaveProfile(ctx, profile); err != nil {
		return domain.Profile{}, err
	}
	return profile, nil
}

func (s Service) Get(ctx context.Context, userID string) (domain.Profile, error) {
	profile, err := s.repository.GetProfile(ctx, userID)
	if err != nil {
		profile = domain.Profile{
			UserID:         userID,
			Level:          1,
			CurrentLevelXP: 0,
			NextLevelXP:    domain.XPNeededForLevel(1),
			Stats: map[domain.Stat]int{
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
			},
		}
	}
	return profile, nil
}
