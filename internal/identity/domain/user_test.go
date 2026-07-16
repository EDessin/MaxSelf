package domain

import "testing"

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Demo@MaxSelf.APP  "); got != "demo@maxself.app" {
		t.Fatalf("unexpected normalized email: %q", got)
	}
}
