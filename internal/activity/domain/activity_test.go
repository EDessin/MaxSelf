package domain

import (
	"errors"
	"testing"
)

func TestRulesAndRuleFor(t *testing.T) {
	rules := Rules()
	if len(rules) != 7 {
		t.Fatalf("expected 7 rules, got %d", len(rules))
	}
	if rules[0].Title != "Cardio Session" || rules[0].XP != 30 || rules[0].Stat != StatCardio {
		t.Fatalf("unexpected first rule: %+v", rules[0])
	}

	rule, err := RuleFor(TypeExercise)
	if err != nil {
		t.Fatalf("RuleFor returned error: %v", err)
	}
	if rule.Title != "Move Your Body" || rule.XP != 40 || rule.Stat != StatStrength {
		t.Fatalf("unexpected exercise rule: %+v", rule)
	}

	_, err = RuleFor(ActivityType("unknown"))
	if !errors.Is(err, ErrUnknownActivityType) {
		t.Fatalf("expected ErrUnknownActivityType, got %v", err)
	}
}
