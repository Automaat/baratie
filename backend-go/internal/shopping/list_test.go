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
	items := build(recipes, nil)

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

func TestBuildCrossOffPantry(t *testing.T) {
	recipes := []PlannedRecipe{
		{Name: "Omelette", Ingredients: []string{"3 eggs", "butter", "chives"}},
	}
	items := build(recipes, []string{"Egg", "  BUTTER  "})

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
	if items := build(nil, nil); len(items) != 0 {
		t.Fatalf("items = %v, want empty", items)
	}
	// Blank ingredient lines are dropped.
	items := build([]PlannedRecipe{{Name: "X", Ingredients: []string{"", "  "}}}, nil)
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
