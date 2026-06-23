package recipes

import (
	"strings"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
)

// validate checks the request fields and normalizes the slices in place,
// returning the first failure as a 422-shaped ValidationError.
func validate(req *createRequest) *httputil.ValidationError {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
	}
	if len(req.Name) > 200 {
		return &httputil.ValidationError{Field: "name", Msg: "Name too long (max 200 characters)"}
	}
	if req.Servings < 1 {
		return &httputil.ValidationError{Field: "servings", Msg: "Servings must be at least 1"}
	}
	if req.PrepMinutes < 0 {
		return &httputil.ValidationError{Field: "prep_minutes", Msg: "Prep minutes cannot be negative"}
	}
	if req.CookMinutes < 0 {
		return &httputil.ValidationError{Field: "cook_minutes", Msg: "Cook minutes cannot be negative"}
	}
	if vErr := validateMacros(req); vErr != nil {
		return vErr
	}
	req.Ingredients = cleanStrings(req.Ingredients)
	req.Tags = cleanStrings(req.Tags)
	return nil
}

// validateMacros rejects negative per-serving nutrition values.
func validateMacros(req *createRequest) *httputil.ValidationError {
	macros := []struct {
		field string
		value float64
		label string
	}{
		{"calories_kcal", req.CaloriesKcal, "Calories"},
		{"protein_g", req.ProteinG, "Protein"},
		{"carbs_g", req.CarbsG, "Carbs"},
		{"fat_g", req.FatG, "Fat"},
	}
	for _, m := range macros {
		if m.value < 0 {
			return &httputil.ValidationError{Field: m.field, Msg: m.label + " cannot be negative"}
		}
	}
	return nil
}

// cleanStrings trims each entry and drops the blanks. Returns a non-nil empty
// slice so the column stores `{}` rather than NULL.
func cleanStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
