package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/progress/domain"
)

type fakeProgressRepository struct {
	profile      domain.Profile
	getErr       error
	saveErr      error
	savedProfile domain.Profile
}

func (r *fakeProgressRepository) GetProfile(_ context.Context, _ string) (domain.Profile, error) {
	return r.profile, r.getErr
}

func (r *fakeProgressRepository) SaveProfile(_ context.Context, profile domain.Profile) error {
	r.savedProfile = profile
	return r.saveErr
}

func TestAwardUpdatesExistingProfile(t *testing.T) {
	last := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	occurredAt := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	repo := &fakeProgressRepository{
		profile: domain.Profile{
			UserID:           "user-1",
			Level:            1,
			TotalXP:          90,
			CurrentLevelXP:   90,
			NextLevelXP:      100,
			StreakDays:       2,
			LastActivityDate: &last,
			Stats:            map[domain.Stat]int{domain.StatStrength: 10},
		},
	}

	profile, err := NewService(repo).Award(context.Background(), domain.Award{
		UserID:     "user-1",
		ActivityID: "activity-1",
		XP:         40,
		Stat:       domain.StatStrength,
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("Award returned error: %v", err)
	}

	if profile.TotalXP != 130 || profile.Level != 2 || profile.CurrentLevelXP != 30 || profile.NextLevelXP != 200 {
		t.Fatalf("unexpected leveled profile: %+v", profile)
	}
	if profile.Stats[domain.StatStrength] != 50 || profile.Stats[domain.StatConsistency] != 5 {
		t.Fatalf("unexpected stats: %+v", profile.Stats)
	}
	if profile.StreakDays != 3 {
		t.Fatalf("expected streak 3, got %d", profile.StreakDays)
	}
	if repo.savedProfile.TotalXP != profile.TotalXP {
		t.Fatalf("profile was not saved: %+v", repo.savedProfile)
	}
}

func TestAwardCreatesFallbackProfileAndPropagatesSaveError(t *testing.T) {
	saveErr := errors.New("save failed")
	repo := &fakeProgressRepository{getErr: errors.New("not found"), saveErr: saveErr}

	_, err := NewService(repo).Award(context.Background(), domain.Award{
		UserID:     "user-1",
		XP:         10,
		Stat:       domain.StatFuel,
		OccurredAt: time.Now(),
	})
	if !errors.Is(err, saveErr) {
		t.Fatalf("expected save error, got %v", err)
	}
	if repo.savedProfile.UserID != "user-1" || repo.savedProfile.Stats[domain.StatFuel] != 10 {
		t.Fatalf("fallback profile was not built before save: %+v", repo.savedProfile)
	}
}

func TestGetReturnsStoredOrDefaultProfile(t *testing.T) {
	stored := domain.Profile{UserID: "user-1", Level: 4, Stats: map[domain.Stat]int{}}
	repo := &fakeProgressRepository{profile: stored}

	profile, err := NewService(repo).Get(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if profile.Level != 4 {
		t.Fatalf("expected stored profile, got %+v", profile)
	}

	repo.getErr = errors.New("not found")
	profile, err = NewService(repo).Get(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("Get fallback returned error: %v", err)
	}
	if profile.UserID != "user-2" || profile.Level != 1 || profile.NextLevelXP != 100 {
		t.Fatalf("unexpected fallback profile: %+v", profile)
	}
	for _, stat := range []domain.Stat{domain.StatStrength, domain.StatFuel, domain.StatRecovery, domain.StatMindset, domain.StatConsistency} {
		if profile.Stats[stat] != 0 {
			t.Fatalf("expected default stat %s to be zero, got %+v", stat, profile.Stats)
		}
	}
}
