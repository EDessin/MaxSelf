package domain

import (
	"testing"
	"time"
)

func TestLevelFor(t *testing.T) {
	tests := []struct {
		name           string
		totalXP        int
		level          int
		currentLevelXP int
		nextLevelXP    int
	}{
		{name: "start", totalXP: 0, level: 1, currentLevelXP: 0, nextLevelXP: 100},
		{name: "within first level", totalXP: 99, level: 1, currentLevelXP: 99, nextLevelXP: 100},
		{name: "second level", totalXP: 100, level: 2, currentLevelXP: 0, nextLevelXP: 200},
		{name: "third level progress", totalXP: 350, level: 3, currentLevelXP: 50, nextLevelXP: 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, currentLevelXP, nextLevelXP := LevelFor(tt.totalXP)
			if level != tt.level || currentLevelXP != tt.currentLevelXP || nextLevelXP != tt.nextLevelXP {
				t.Fatalf("LevelFor(%d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.totalXP, level, currentLevelXP, nextLevelXP, tt.level, tt.currentLevelXP, tt.nextLevelXP)
			}
		})
	}
}

func TestXPNeededForLevelNormalizesLowLevels(t *testing.T) {
	if got := XPNeededForLevel(0); got != 100 {
		t.Fatalf("expected level 0 to need 100 XP, got %d", got)
	}
	if got := XPNeededForLevel(3); got != 300 {
		t.Fatalf("expected level 3 to need 300 XP, got %d", got)
	}
}

func TestConsistencyStatFor(t *testing.T) {
	tests := map[Stat]Stat{
		StatCardio:     StatCardioConsistency,
		StatStrength:   StatStrengthConsistency,
		StatFuel:       StatFuelConsistency,
		StatRecovery:   StatRecoveryConsistency,
		StatMindset:    StatMindsetConsistency,
		StatBiometrics: StatBiometricsConsistency,
		Stat("other"):  "",
	}

	for stat, want := range tests {
		if got := ConsistencyStatFor(stat); got != want {
			t.Fatalf("ConsistencyStatFor(%s) = %s, want %s", stat, got, want)
		}
	}
}

func TestDefaultStatsIncludesTrackedStats(t *testing.T) {
	stats := DefaultStats()
	for _, stat := range []Stat{
		StatCardio,
		StatStrength,
		StatFuel,
		StatRecovery,
		StatMindset,
		StatBiometrics,
		StatCardioConsistency,
		StatStrengthConsistency,
		StatFuelConsistency,
		StatRecoveryConsistency,
		StatMindsetConsistency,
		StatBiometricsConsistency,
	} {
		if xp, ok := stats[stat]; !ok || xp != 0 {
			t.Fatalf("expected default stat %s to be present at zero, got %+v", stat, stats)
		}
	}
}

func TestUpdatedStreak(t *testing.T) {
	base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

	streak, date := UpdatedStreak(0, nil, base)
	if streak != 1 || date == nil || !date.Equal(time.Date(2026, 7, 16, 0, 0, 0, 0, base.Location())) {
		t.Fatalf("unexpected initial streak: %d %v", streak, date)
	}

	streak, date = UpdatedStreak(4, date, base.Add(2*time.Hour))
	if streak != 4 {
		t.Fatalf("same day should keep streak, got %d", streak)
	}

	streak, date = UpdatedStreak(4, date, base.AddDate(0, 0, 1))
	if streak != 5 {
		t.Fatalf("next day should increment streak, got %d", streak)
	}

	streak, _ = UpdatedStreak(5, date, base.AddDate(0, 0, 3))
	if streak != 1 {
		t.Fatalf("missed day should reset streak, got %d", streak)
	}
}
