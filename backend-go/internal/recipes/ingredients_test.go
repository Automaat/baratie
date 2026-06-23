package recipes

import "testing"

func TestComputePerServing(t *testing.T) {
	// 200 g chicken (165 kcal/100g, 31 protein) + 100 g rice (130 kcal, 2.7p),
	// recipe makes 2 servings.
	ings := []StructuredIngredient{
		{Amount: 200, Unit: "g", KcalPer100g: 165, ProteinPer100g: 31},
		{Amount: 100, Unit: "g", KcalPer100g: 130, ProteinPer100g: 2.7},
	}
	cm, usable := computePerServing(ings, 2)
	if !usable {
		t.Fatal("expected usable macros")
	}
	// total kcal = 330 + 130 = 460; per serving = 230.
	if cm.Kcal != 230 {
		t.Fatalf("kcal/serving = %v, want 230", cm.Kcal)
	}
	// total protein = 62 + 2.7 = 64.7; per serving = 32.35.
	if cm.Protein < 32.34 || cm.Protein > 32.36 {
		t.Fatalf("protein/serving = %v, want ~32.35", cm.Protein)
	}
}

func TestComputePerServingKgConversion(t *testing.T) {
	// 0.5 kg = 500 g of a 200 kcal/100g food → 1000 kcal total, 1 serving.
	ings := []StructuredIngredient{{Amount: 0.5, Unit: "kg", KcalPer100g: 200}}
	cm, usable := computePerServing(ings, 1)
	if !usable || cm.Kcal != 1000 {
		t.Fatalf("kcal = %v (usable=%v), want 1000", cm.Kcal, usable)
	}
}

func TestComputePerServingNotUsable(t *testing.T) {
	// Zero-macro food (migration default) and a non-mass unit yield nothing.
	ings := []StructuredIngredient{
		{Amount: 100, Unit: "g", KcalPer100g: 0},
		{Amount: 2, Unit: "szt", KcalPer100g: 80},
	}
	if _, usable := computePerServing(ings, 1); usable {
		t.Fatal("expected not usable (no derivable macros)")
	}
}

func TestComputePerServingZeroServingsGuarded(t *testing.T) {
	ings := []StructuredIngredient{{Amount: 100, Unit: "g", KcalPer100g: 50}}
	if cm, _ := computePerServing(ings, 0); cm.Kcal != 50 {
		t.Fatalf("kcal = %v, want 50 (servings floored to 1)", cm.Kcal)
	}
}
