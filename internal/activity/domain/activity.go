package domain

import (
	"errors"
	"time"
)

type ActivityType string
type Stat string

const (
	TypeExercise    ActivityType = "exercise"
	TypeHealthyMeal ActivityType = "healthy_meal"
	TypeHydration   ActivityType = "hydration"
	TypeSleep       ActivityType = "sleep"
	TypeMindfulness ActivityType = "mindfulness"
	TypeRecovery    ActivityType = "recovery"

	StatStrength    Stat = "strength"
	StatFuel        Stat = "fuel"
	StatRecovery    Stat = "recovery"
	StatMindset     Stat = "mindset"
	StatConsistency Stat = "consistency"
)

var ErrUnknownActivityType = errors.New("unknown activity type")

type Activity struct {
	ID         string
	UserID     string
	Type       ActivityType
	Title      string
	Notes      string
	XP         int
	Stat       Stat
	OccurredAt time.Time
	CreatedAt  time.Time
}

type ActivityRule struct {
	Type  ActivityType `json:"type"`
	Title string       `json:"title"`
	XP    int          `json:"xp"`
	Stat  Stat         `json:"stat"`
	Icon  string       `json:"icon"`
	Color string       `json:"color"`
}

func Rules() []ActivityRule {
	return []ActivityRule{
		{Type: TypeExercise, Title: "Move Your Body", XP: 40, Stat: StatStrength, Icon: "dumbbell", Color: "#ff5a5f"},
		{Type: TypeHealthyMeal, Title: "Nourishing Meal", XP: 25, Stat: StatFuel, Icon: "apple", Color: "#22c55e"},
		{Type: TypeHydration, Title: "Hydration Boost", XP: 10, Stat: StatFuel, Icon: "droplet", Color: "#38bdf8"},
		{Type: TypeSleep, Title: "Sleep Goal Met", XP: 35, Stat: StatRecovery, Icon: "moon", Color: "#6366f1"},
		{Type: TypeMindfulness, Title: "Mindset Moment", XP: 20, Stat: StatMindset, Icon: "sparkles", Color: "#a855f7"},
		{Type: TypeRecovery, Title: "Recovery Ritual", XP: 20, Stat: StatRecovery, Icon: "heart-pulse", Color: "#14b8a6"},
	}
}

func RuleFor(activityType ActivityType) (ActivityRule, error) {
	for _, rule := range Rules() {
		if rule.Type == activityType {
			return rule, nil
		}
	}
	return ActivityRule{}, ErrUnknownActivityType
}
