package domain

import (
	"errors"
	"testing"
)

func TestRulesAndRuleFor(t *testing.T) {
	rules := Rules()
	if len(rules) != 9 {
		t.Fatalf("expected 9 rules, got %d", len(rules))
	}
	if rules[0].Title != "Cardio Session" || rules[0].XP != 30 || rules[0].Stat != StatCardio {
		t.Fatalf("unexpected first rule: %+v", rules[0])
	}
	if rules[0].Color != "#f59e0b" {
		t.Fatalf("expected cardio to use consistency yellow, got %s", rules[0].Color)
	}

	rule, err := RuleFor(TypeExercise)
	if err != nil {
		t.Fatalf("RuleFor returned error: %v", err)
	}
	if rule.Title != "Strength Session" || rule.XP != 40 || rule.Stat != StatStrength {
		t.Fatalf("unexpected exercise rule: %+v", rule)
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
