package recipes

import "testing"

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name string
		req  createRequest
	}{
		{"empty name", createRequest{Name: "  ", Servings: 1}},
		{"zero servings", createRequest{Name: "Soup", Servings: 0}},
		{"negative prep", createRequest{Name: "Soup", Servings: 1, PrepMinutes: -1}},
		{"negative cook", createRequest{Name: "Soup", Servings: 1, CookMinutes: -5}},
		{"negative calories", createRequest{Name: "Soup", Servings: 1, CaloriesKcal: -1}},
		{"negative protein", createRequest{Name: "Soup", Servings: 1, ProteinG: -0.5}},
		{"negative carbs", createRequest{Name: "Soup", Servings: 1, CarbsG: -2}},
		{"negative fat", createRequest{Name: "Soup", Servings: 1, FatG: -3}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := c.req
			if validate(&req) == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateNormalizes(t *testing.T) {
	req := createRequest{
		Name:         "  Tomato Soup  ",
		Servings:     2,
		Ingredients:  []string{"  tomato  ", "", "  basil "},
		Tags:         []string{" quick ", "  "},
		CaloriesKcal: 320,
		ProteinG:     12.5,
	}
	if vErr := validate(&req); vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if req.Name != "Tomato Soup" {
		t.Fatalf("name = %q, want trimmed", req.Name)
	}
	if req.CaloriesKcal != 320 || req.ProteinG != 12.5 {
		t.Fatalf("macros not preserved: %v / %v", req.CaloriesKcal, req.ProteinG)
	}
	if len(req.Ingredients) != 2 || req.Ingredients[0] != "tomato" || req.Ingredients[1] != "basil" {
		t.Fatalf("ingredients = %v, want cleaned", req.Ingredients)
	}
	if len(req.Tags) != 1 || req.Tags[0] != "quick" {
		t.Fatalf("tags = %v, want cleaned", req.Tags)
	}
}
