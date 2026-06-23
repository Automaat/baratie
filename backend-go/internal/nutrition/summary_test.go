package nutrition

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestSummarizeGroupsByDay(t *testing.T) {
	contribs := []Contribution{
		{Date: day("2026-06-01"), CaloriesKcal: 500, ProteinG: 40, CarbsG: 50, FatG: 10},
		{Date: day("2026-06-01"), CaloriesKcal: 700, ProteinG: 50, CarbsG: 60, FatG: 20},
		{Date: day("2026-06-02"), CaloriesKcal: 300, ProteinG: 20, CarbsG: 30, FatG: 5},
	}
	s := summarize(contribs, nil)

	if len(s.Days) != 2 {
		t.Fatalf("days = %d, want 2", len(s.Days))
	}
	if s.Days[0].Date.Format("2006-01-02") != "2026-06-01" {
		t.Fatalf("days not chronological: %v", s.Days[0].Date)
	}
	if s.Days[0].Total.CaloriesKcal != 1200 || s.Days[0].Meals != 2 {
		t.Fatalf("day 1 = %+v, want 1200 kcal / 2 meals", s.Days[0])
	}
	if s.Totals.CaloriesKcal != 1500 || s.Totals.ProteinG != 110 {
		t.Fatalf("totals = %+v, want 1500 kcal / 110 protein", s.Totals)
	}
	if s.Meals != 3 {
		t.Fatalf("meals = %d, want 3", s.Meals)
	}
	// Average is per active day (2 days): 1500/2 = 750 kcal.
	if s.Average.CaloriesKcal != 750 || s.Average.ProteinG != 55 {
		t.Fatalf("average = %+v, want 750 kcal / 55 protein", s.Average)
	}
	if s.Targets != nil {
		t.Fatalf("targets should be nil, got %+v", s.Targets)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := summarize(nil, nil)
	if len(s.Days) != 0 {
		t.Fatalf("days = %d, want 0", len(s.Days))
	}
	if s.Totals != (Macros{}) || s.Average != (Macros{}) {
		t.Fatalf("empty summary should be zero: totals=%+v average=%+v", s.Totals, s.Average)
	}
	if s.Meals != 0 {
		t.Fatalf("meals = %d, want 0", s.Meals)
	}
}

func TestSummarizeCarriesTargets(t *testing.T) {
	targets := &Macros{CaloriesKcal: 2000, ProteinG: 170, CarbsG: 200, FatG: 60}
	contribs := []Contribution{
		{Date: day("2026-06-01"), CaloriesKcal: 1800, ProteinG: 150, CarbsG: 180, FatG: 55},
	}
	s := summarize(contribs, targets)
	if s.Targets == nil || *s.Targets != *targets {
		t.Fatalf("targets not carried: %+v", s.Targets)
	}
	// The handler diffs per day: 1800-2000 = -200 kcal (under), 150-170 = -20 protein.
	delta := s.Days[0].Total.sub(*s.Targets)
	if delta.CaloriesKcal != -200 || delta.ProteinG != -20 {
		t.Fatalf("delta = %+v, want -200 kcal / -20 protein", delta)
	}
}

func TestMacrosArithmetic(t *testing.T) {
	a := Macros{CaloriesKcal: 100, ProteinG: 10, CarbsG: 20, FatG: 5}
	b := Macros{CaloriesKcal: 40, ProteinG: 4, CarbsG: 8, FatG: 1}
	if sum := a.add(b); sum.CaloriesKcal != 140 || sum.ProteinG != 14 {
		t.Fatalf("add = %+v", sum)
	}
	if diff := a.sub(b); diff.CaloriesKcal != 60 || diff.FatG != 4 {
		t.Fatalf("sub = %+v", diff)
	}
	if sc := a.scale(2); sc.CaloriesKcal != 50 || sc.CarbsG != 10 {
		t.Fatalf("scale = %+v", sc)
	}
}
