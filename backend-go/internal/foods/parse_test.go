package foods

import "testing"

func TestParseIngredientLine(t *testing.T) {
	cases := []struct {
		in     string
		name   string
		amount float64
		unit   string
	}{
		{"200 g chicken breast", "chicken breast", 200, "g"},
		{"200g chicken", "chicken", 200, "g"},
		{"1.5 kg potatoes", "potatoes", 1.5, "kg"},
		{"2 eggs", "eggs", 2, "szt"},
		{"250 ml milk", "milk", 250, "ml"},
		{"salt", "salt", 0, "szt"},
		{"  Olive Oil  ", "olive oil", 0, "szt"},
		{"2,5 dl cream", "dl cream", 2.5, "szt"}, // unknown unit kept in name
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := parseIngredientLine(c.in)
			if got.Name != c.name || got.Amount != c.amount || got.Unit != c.unit {
				t.Fatalf("parse(%q) = %+v, want {%q %v %q}", c.in, got, c.name, c.amount, c.unit)
			}
		})
	}
}

func TestParseIngredientLineSkips(t *testing.T) {
	for _, in := range []string{"", "   ", "200 g"} {
		if got := parseIngredientLine(in); got.Name != "" {
			t.Fatalf("parse(%q) = %+v, want empty name (skip)", in, got)
		}
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]createRequest{
		"empty name":       {Name: "  "},
		"negative kcal":    {Name: "Egg", KcalPer100g: -1},
		"negative protein": {Name: "Egg", ProteinPer100g: -2},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			r := req
			if validate(&r) == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateNormalizes(t *testing.T) {
	req := createRequest{Name: "  Chicken  ", KcalPer100g: 165, ProteinPer100g: 31}
	if vErr := validate(&req); vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if req.Name != "Chicken" {
		t.Fatalf("name = %q, want trimmed", req.Name)
	}
}
