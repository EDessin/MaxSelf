package domain

import (
	"errors"
	"testing"
)

func TestRulesAndRuleFor(t *testing.T) {
	rules := Rules()
	if len(rules) != 17 {
		t.Fatalf("expected 17 rules, got %d", len(rules))
	}
	if rules[0].Title != "Cardio Session" || rules[0].XP != 30 || rules[0].Stat != StatCardio {
		t.Fatalf("unexpected first rule: %+v", rules[0])
	}
	if rules[0].Color != "#f59e0b" {
		t.Fatalf("expected cardio to use consistency yellow, got %s", rules[0].Color)
	}
	for _, rule := range rules {
		if rule.Color != CategoryColor(rule.Stat) {
			t.Fatalf("expected %s to use %s category color %s, got %s", rule.Type, rule.Stat, CategoryColor(rule.Stat), rule.Color)
		}
		if rule.Goal == "" {
			t.Fatalf("expected %s to define a goal", rule.Type)
		}
	}

	rule, err := RuleFor(TypeDailyStepsBronze)
	if err != nil {
		t.Fatalf("RuleFor returned error for daily steps bronze: %v", err)
	}
	if rule.Title != "Daily Steps — Bronze" || rule.XP != 20 || rule.Stat != StatCardio || rule.Icon != "footprints" || rule.Goal != "6000 steps" || rule.Tier != "Bronze" || rule.ThresholdValue != 6000 || rule.ThresholdUnit != "steps" || rule.FollowUpType != TypeDailyStepsSilver {
		t.Fatalf("unexpected daily steps bronze rule: %+v", rule)
	}

	rule, err = RuleFor(TypeDailyStepsGold)
	if err != nil {
		t.Fatalf("RuleFor returned error for daily steps gold: %v", err)
	}
	if rule.XP != 45 || rule.PrerequisiteType != TypeDailyStepsSilver || rule.FollowUpType != TypeDailyStepsDiamond {
		t.Fatalf("unexpected daily steps gold rule: %+v", rule)
	}

	rule, err = RuleFor(TypeHydrationDiamond)
	if err != nil {
		t.Fatalf("RuleFor returned error for hydration diamond: %v", err)
	}
	if rule.Title != "Hydration Boost — Diamond" || rule.XP != 30 || rule.ThresholdValue != 2000 || rule.ThresholdUnit != "ml" || rule.PrerequisiteType != TypeHydrationGold {
		t.Fatalf("unexpected hydration diamond rule: %+v", rule)
	}

	rule, err = RuleFor(TypeSleep)
	if err != nil {
		t.Fatalf("RuleFor returned error for sleep: %v", err)
	}
	if rule.Title != "Sleep Goal Met" || rule.XP != 35 || rule.Stat != StatRecovery || rule.Goal != "7 hours" || rule.ThresholdValue != SleepGoalHours || rule.ThresholdUnit != "hours" {
		t.Fatalf("unexpected sleep rule: %+v", rule)
	}

	rule, err = RuleFor(TypeMindfulness)
	if err != nil {
		t.Fatalf("RuleFor returned error for mindfulness: %v", err)
	}
	if rule.Goal != "not ready yet" {
		t.Fatalf("unexpected mindfulness goal: %+v", rule)
	}

	rule, err = RuleFor(TypeExercise)
	if err != nil {
		t.Fatalf("RuleFor returned error: %v", err)
	}
	if rule.Title != "Resistance Training" || rule.XP != 40 || rule.Stat != StatStrength || rule.Goal != "10+ min resistance training" {
		t.Fatalf("unexpected exercise rule: %+v", rule)
	}

	rule, err = RuleFor(TypeMobility)
	if err != nil {
		t.Fatalf("RuleFor returned error for mobility: %v", err)
	}
	if rule.Title != "Mobility Session" || rule.XP != 20 || rule.Stat != StatStrength || rule.Icon != "person-standing" || rule.Goal != "10+ min mobility" {
		t.Fatalf("unexpected mobility rule: %+v", rule)
	}

	rule, err = RuleFor(TypeScaleMeasurement)
	if err != nil {
		t.Fatalf("RuleFor returned error for scale measurement: %v", err)
	}
	if rule.Title != "Scale Measurement" || rule.XP != 15 || rule.Stat != StatBiometrics || rule.Icon != "scale" || rule.Color != "#0891b2" {
		t.Fatalf("unexpected scale measurement rule: %+v", rule)
	}

	rule, err = RuleFor(TypeWaistToHeightRatio)
	if err != nil {
		t.Fatalf("RuleFor returned error for waist-to-height ratio: %v", err)
	}
	if rule.Title != "Waist-to-Height Ratio" || rule.XP != 15 || rule.Stat != StatBiometrics || rule.Icon != "ruler" || rule.Color != "#0891b2" {
		t.Fatalf("unexpected waist-to-height ratio rule: %+v", rule)
	}

	_, err = RuleFor(ActivityType("unknown"))
	if !errors.Is(err, ErrUnknownActivityType) {
		t.Fatalf("expected ErrUnknownActivityType, got %v", err)
	}
}
