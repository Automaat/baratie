package nutrition

import (
	"net/url"
	"testing"
)

func TestParseTargetsAbsent(t *testing.T) {
	_, present, vErr := parseTargets(url.Values{})
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if present {
		t.Fatal("present should be false when no target params given")
	}
}

func TestParseTargetsPartialDefaultsZero(t *testing.T) {
	q := url.Values{"target_protein_g": {"170"}}
	m, present, vErr := parseTargets(q)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !present {
		t.Fatal("present should be true when a target is given")
	}
	if m.ProteinG != 170 {
		t.Fatalf("protein = %v, want 170", m.ProteinG)
	}
	if m.CaloriesKcal != 0 || m.CarbsG != 0 || m.FatG != 0 {
		t.Fatalf("unset targets should default to 0: %+v", m)
	}
}

func TestParseTargetsRejectsBad(t *testing.T) {
	cases := map[string]url.Values{
		"negative":   {"target_kcal": {"-1"}},
		"non-number": {"target_fat_g": {"abc"}},
	}
	for name, q := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, vErr := parseTargets(q)
			if vErr == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
