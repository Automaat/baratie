package shopping

import (
	"reflect"
	"testing"
)

func TestBuildDedupesAndAttributesRecipes(t *testing.T) {
	recipes := []PlannedRecipe{
		{Name: "Soup", Ingredients: []string{"Tomato", "  Basil ", "salt"}},
		{Name: "Salad", Ingredients: []string{"tomato", "Olive oil"}},
	}
	items := build(nil, recipes, nil)

	// "Tomato"/"tomato" collapse to one entry, keeping the first-seen display.
	var tomato *Item
	for i := range items {
		if items[i].Name == "Tomato" {
			tomato = &items[i]
		}
	}
	if tomato == nil {
		t.Fatalf("tomato not in list: %+v", items)
	}
	if !reflect.DeepEqual(tomato.Recipes, []string{"Salad", "Soup"}) {
		t.Fatalf("tomato recipes = %v, want [Salad Soup]", tomato.Recipes)
	}
	// 5 ingredient lines collapse to 4 distinct: tomato, basil, salt, olive oil.
	if len(items) != 4 {
		t.Fatalf("items = %d (%v), want 4 distinct", len(items), items)
	}
	// Sorted by normalized key: basil, olive oil, salt, tomato.
	if items[0].Name != "Basil" || items[3].Name != "Tomato" {
		t.Fatalf("not sorted by normalized key: %v", names(items))
	}
}

func TestBuildStructuredSumsPerFoodUnit(t *testing.T) {
	lines := []StructuredLine{
		{Recipe: "Bowl", Food: "Chicken", Amount: 200, Unit: "g"},
		{Recipe: "Wrap", Food: "chicken", Amount: 150, Unit: "g"},
		{Recipe: "Bowl", Food: "Rice", Amount: 100, Unit: "g"},
		{Recipe: "Snack", Food: "Egg", Amount: 2, Unit: "szt"},
	}
	items := build(lines, nil, nil)

	var chicken *Item
	for i := range items {
		if items[i].Name == "Chicken" {
			chicken = &items[i]
		}
	}
	if chicken == nil {
		t.Fatalf("chicken not in list: %+v", items)
	}
	// 200 g + 150 g summed across recipes (case-insensitive food name).
	if chicken.Amount != 350 || chicken.Unit != "g" {
		t.Fatalf("chicken = %v %s, want 350 g", chicken.Amount, chicken.Unit)
	}
	if !reflect.DeepEqual(chicken.Recipes, []string{"Bowl", "Wrap"}) {
		t.Fatalf("chicken recipes = %v, want [Bowl Wrap]", chicken.Recipes)
	}
	if len(items) != 3 {
		t.Fatalf("items = %d (%v), want 3 (chicken, rice, egg)", len(items), items)
	}
}

func TestBuildStructuredSeparatesByUnit(t *testing.T) {
	lines := []StructuredLine{
		{Recipe: "A", Food: "Milk", Amount: 200, Unit: "ml"},
		{Recipe: "B", Food: "Milk", Amount: 1, Unit: "l"},
	}
	items := build(lines, nil, nil)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2 (ml and l are distinct units)", len(items))
	}
}

func TestBuildCrossOffPantry(t *testing.T) {
	recipes := []PlannedRecipe{
		{Name: "Omelette", Ingredients: []string{"3 eggs", "butter", "chives"}},
	}
	items := build(nil, recipes, []string{"Egg", "  BUTTER  "})

	got := map[string]bool{}
	for _, it := range items {
		got[it.Name] = it.InPantry
	}
	if !got["3 eggs"] {
		t.Fatal("'3 eggs' should be flagged in pantry (contains 'egg')")
	}
	if !got["butter"] {
		t.Fatal("'butter' should be flagged in pantry")
	}
	if got["chives"] {
		t.Fatal("'chives' should not be in pantry")
	}
}

func TestBuildEmpty(t *testing.T) {
	if items := build(nil, nil, nil); len(items) != 0 {
		t.Fatalf("items = %v, want empty", items)
	}
	// Blank ingredient lines are dropped.
	items := build(nil, []PlannedRecipe{{Name: "X", Ingredients: []string{"", "  "}}}, nil)
	if len(items) != 0 {
		t.Fatalf("blank ingredients should be dropped, got %v", items)
	}
}

func names(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
}
