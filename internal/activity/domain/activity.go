package domain

import (
	"errors"
	"time"
)

type ActivityType string
type Stat string

const (
	TypeExercise           ActivityType = "exercise"
	TypeCardio             ActivityType = "cardio"
	TypeDailyStepsBronze   ActivityType = "daily_steps_bronze"
	TypeDailyStepsSilver   ActivityType = "daily_steps_silver"
	TypeDailyStepsGold     ActivityType = "daily_steps_gold"
	TypeDailyStepsDiamond  ActivityType = "daily_steps_diamond"
	TypeHealthyMeal        ActivityType = "healthy_meal"
	TypeHydrationBronze    ActivityType = "hydration_bronze"
	TypeHydrationSilver    ActivityType = "hydration_silver"
	TypeHydrationGold      ActivityType = "hydration_gold"
	TypeHydrationDiamond   ActivityType = "hydration_diamond"
	TypeSleep              ActivityType = "sleep"
	TypeMindfulness        ActivityType = "mindfulness"
	TypeRecovery           ActivityType = "recovery"
	TypeScaleMeasurement   ActivityType = "scale_measurement"
	TypeWaistToHeightRatio ActivityType = "waist_to_height_ratio"

	StatCardio      Stat = "cardio"
	StatStrength    Stat = "strength"
	StatFuel        Stat = "fuel"
	StatRecovery    Stat = "recovery"
	StatMindset     Stat = "mindset"
	StatBiometrics  Stat = "biometrics"
	StatConsistency Stat = "consistency"
)

var ErrUnknownActivityType = errors.New("unknown activity type")

const (
	ColorCardio     = "#f59e0b"
	ColorStrength   = "#ff5a5f"
	ColorFuel       = "#22c55e"
	ColorRecovery   = "#6366f1"
	ColorMindset    = "#a855f7"
	ColorBiometrics = "#0891b2"

	SleepGoalHours = 7
)

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
	Type             ActivityType `json:"type"`
	Title            string       `json:"title"`
	XP               int          `json:"xp"`
	Stat             Stat         `json:"stat"`
	Icon             string       `json:"icon"`
	Color            string       `json:"color"`
	Goal             string       `json:"goal,omitempty"`
	Tier             string       `json:"tier,omitempty"`
	ThresholdValue   int          `json:"thresholdValue,omitempty"`
	ThresholdUnit    string       `json:"thresholdUnit,omitempty"`
	FollowUpType     ActivityType `json:"followUpType,omitempty"`
	PrerequisiteType ActivityType `json:"prerequisiteType,omitempty"`
}

func Rules() []ActivityRule {
	return applyCategoryColors([]ActivityRule{
		{Type: TypeCardio, Title: "Cardio Session", XP: 30, Stat: StatCardio, Icon: "flame", Goal: "10+ min cardio"},
		{Type: TypeDailyStepsBronze, Title: "Daily Steps — Bronze", XP: 20, Stat: StatCardio, Icon: "footprints", Goal: "6000 steps", Tier: "Bronze", ThresholdValue: 6000, ThresholdUnit: "steps", FollowUpType: TypeDailyStepsSilver},
		{Type: TypeDailyStepsSilver, Title: "Daily Steps — Silver", XP: 30, Stat: StatCardio, Icon: "footprints", Goal: "8000 steps", Tier: "Silver", ThresholdValue: 8000, ThresholdUnit: "steps", FollowUpType: TypeDailyStepsGold, PrerequisiteType: TypeDailyStepsBronze},
		{Type: TypeDailyStepsGold, Title: "Daily Steps — Gold", XP: 45, Stat: StatCardio, Icon: "footprints", Goal: "10000 steps", Tier: "Gold", ThresholdValue: 10000, ThresholdUnit: "steps", FollowUpType: TypeDailyStepsDiamond, PrerequisiteType: TypeDailyStepsSilver},
		{Type: TypeDailyStepsDiamond, Title: "Daily Steps — Diamond", XP: 70, Stat: StatCardio, Icon: "footprints", Goal: "15000 steps", Tier: "Diamond", ThresholdValue: 15000, ThresholdUnit: "steps", PrerequisiteType: TypeDailyStepsGold},
		{Type: TypeExercise, Title: "Strength Session", XP: 40, Stat: StatStrength, Icon: "dumbbell", Goal: "10+ min strength"},
		{Type: TypeHealthyMeal, Title: "Nourishing Meal", XP: 25, Stat: StatFuel, Icon: "apple", Goal: "Log nutrition"},
		{Type: TypeHydrationBronze, Title: "Hydration Boost — Bronze", XP: 10, Stat: StatFuel, Icon: "droplet", Goal: "500 ml", Tier: "Bronze", ThresholdValue: 500, ThresholdUnit: "ml", FollowUpType: TypeHydrationSilver},
		{Type: TypeHydrationSilver, Title: "Hydration Boost — Silver", XP: 15, Stat: StatFuel, Icon: "droplet", Goal: "1000 ml", Tier: "Silver", ThresholdValue: 1000, ThresholdUnit: "ml", FollowUpType: TypeHydrationGold, PrerequisiteType: TypeHydrationBronze},
		{Type: TypeHydrationGold, Title: "Hydration Boost — Gold", XP: 20, Stat: StatFuel, Icon: "droplet", Goal: "1500 ml", Tier: "Gold", ThresholdValue: 1500, ThresholdUnit: "ml", FollowUpType: TypeHydrationDiamond, PrerequisiteType: TypeHydrationSilver},
		{Type: TypeHydrationDiamond, Title: "Hydration Boost — Diamond", XP: 30, Stat: StatFuel, Icon: "droplet", Goal: "2000 ml", Tier: "Diamond", ThresholdValue: 2000, ThresholdUnit: "ml", PrerequisiteType: TypeHydrationGold},
		{Type: TypeSleep, Title: "Sleep Goal Met", XP: 35, Stat: StatRecovery, Icon: "moon", Goal: "7 hours", ThresholdValue: SleepGoalHours, ThresholdUnit: "hours"},
		{Type: TypeMindfulness, Title: "Mindset Moment", XP: 20, Stat: StatMindset, Icon: "sparkles", Goal: "not ready yet"},
		{Type: TypeRecovery, Title: "Recovery Ritual", XP: 20, Stat: StatRecovery, Icon: "heart-pulse", Goal: "10+ min mobility"},
		{Type: TypeScaleMeasurement, Title: "Scale Measurement", XP: 15, Stat: StatBiometrics, Icon: "scale", Goal: "Weight or body fat"},
		{Type: TypeWaistToHeightRatio, Title: "Waist-to-Height Ratio", XP: 15, Stat: StatBiometrics, Icon: "ruler", Goal: "Waist + height"},
	})
}

func CategoryColor(stat Stat) string {
	switch stat {
	case StatCardio:
		return ColorCardio
	case StatStrength:
		return ColorStrength
	case StatFuel:
		return ColorFuel
	case StatRecovery:
		return ColorRecovery
	case StatMindset:
		return ColorMindset
	case StatBiometrics:
		return ColorBiometrics
	default:
		return ColorCardio
	}
}

func applyCategoryColors(rules []ActivityRule) []ActivityRule {
	for index := range rules {
		rules[index].Color = CategoryColor(rules[index].Stat)
	}
	return rules
}

func RuleFor(activityType ActivityType) (ActivityRule, error) {
	for _, rule := range Rules() {
		if rule.Type == activityType {
			return rule, nil
		}
	}
	return ActivityRule{}, ErrUnknownActivityType
}
