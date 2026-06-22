package mealplan

import "testing"

func TestValidateRejects(t *testing.T) {
	if validate(&createRequest{PlanDate: "not-a-date", MealType: "lunch"}) == nil {
		t.Fatal("bad date should fail")
	}
	if validate(&createRequest{PlanDate: "2026-01-02", MealType: "brunch"}) == nil {
		t.Fatal("invalid meal type should fail")
	}
}

func TestValidateNormalizes(t *testing.T) {
	id := 4
	req := createRequest{
		PlanDate: "2026-01-02", MealType: "lunch", RecipeID: &id, Note: "  leftovers ",
	}
	if vErr := validate(&req); vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	e := toEntry(&req)
	if e.Note != "leftovers" {
		t.Fatalf("note = %q, want trimmed", e.Note)
	}
	if e.RecipeID == nil || *e.RecipeID != 4 {
		t.Fatalf("recipe id not carried: %v", e.RecipeID)
	}
	if e.PlanDate.Format("2006-01-02") != "2026-01-02" {
		t.Fatalf("date = %v", e.PlanDate)
	}
}
