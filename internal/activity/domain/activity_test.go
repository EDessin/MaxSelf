package domain

import (
	"errors"
	"testing"
)

func TestRulesAndRuleFor(t *testing.T) {
	rules := Rules()
	if len(rules) != 10 {
		t.Fatalf("expected 10 rules, got %d", len(rules))
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

	rule, err = RuleFor(TypeBodyScan)
	if err != nil {
		t.Fatalf("RuleFor returned error for body scan: %v", err)
	}
	if rule.Title != "Body Composition Scan" || rule.XP != 35 || rule.Stat != StatBiometrics || rule.Icon != "scan-line" || rule.Color != "#0891b2" {
		t.Fatalf("unexpected body scan rule: %+v", rule)
	}

	_, err = RuleFor(ActivityType("unknown"))
	if !errors.Is(err, ErrUnknownActivityType) {
		t.Fatalf("expected ErrUnknownActivityType, got %v", err)
	}
}
