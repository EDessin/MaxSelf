package domain

import "time"

type Stat string

const (
	StatCardio              Stat = "cardio"
	StatStrength            Stat = "strength"
	StatFuel                Stat = "fuel"
	StatRecovery            Stat = "recovery"
	StatMindset             Stat = "mindset"
	StatConsistency         Stat = "consistency"
	StatCardioConsistency   Stat = "cardio_consistency"
	StatStrengthConsistency Stat = "strength_consistency"
	StatFuelConsistency     Stat = "fuel_consistency"
	StatRecoveryConsistency Stat = "recovery_consistency"
	StatMindsetConsistency  Stat = "mindset_consistency"
)

type Profile struct {
	UserID           string       `json:"userId"`
	Level            int          `json:"level"`
	TotalXP          int          `json:"totalXp"`
	CurrentLevelXP   int          `json:"currentLevelXp"`
	NextLevelXP      int          `json:"nextLevelXp"`
	StreakDays       int          `json:"streakDays"`
	LastActivityDate *time.Time   `json:"lastActivityDate"`
	Stats            map[Stat]int `json:"stats"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

type Award struct {
	UserID     string
	ActivityID string
	XP         int
	Stat       Stat
	OccurredAt time.Time
}

func ConsistencyStatFor(stat Stat) Stat {
	switch stat {
	case StatCardio:
		return StatCardioConsistency
	case StatStrength:
		return StatStrengthConsistency
	case StatFuel:
		return StatFuelConsistency
	case StatRecovery:
		return StatRecoveryConsistency
	case StatMindset:
		return StatMindsetConsistency
	default:
		return ""
	}
}

func LevelFor(totalXP int) (level int, currentLevelXP int, nextLevelXP int) {
	level = 1
	remaining := totalXP
	for {
		needed := XPNeededForLevel(level)
		if remaining < needed {
			return level, remaining, needed
		}
		remaining -= needed
		level++
	}
}

func XPNeededForLevel(level int) int {
	if level < 1 {
		level = 1
	}
	return level * 100
}

func UpdatedStreak(current int, lastActivityDate *time.Time, occurredAt time.Time) (int, *time.Time) {
	occurredDate := dateOnly(occurredAt)
	if lastActivityDate == nil {
		return 1, &occurredDate
	}
	lastDate := dateOnly(*lastActivityDate)
	if occurredDate.Equal(lastDate) {
		return current, &lastDate
	}
	if occurredDate.Equal(lastDate.AddDate(0, 0, 1)) {
		return current + 1, &occurredDate
	}
	return 1, &occurredDate
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}
