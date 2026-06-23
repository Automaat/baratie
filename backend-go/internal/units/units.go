// Package units holds the small, fixed set of measurement units used by
// structured recipe ingredients and the gram conversion needed to compute
// macros from per-100g food data.
package units

import "strings"

// Canonical is the set of units the app understands. "szt" (piece) has no mass
// conversion — its macro contribution can't be derived from per-100g data.
const (
	Gram       = "g"
	Kilogram   = "kg"
	Milligram  = "mg"
	Milliliter = "ml"
	Liter      = "l"
	Piece      = "szt"
)

// gramsPerUnit maps a mass/volume unit to its weight in grams (volume assumes a
// density of 1 g/ml). Units absent here (e.g. "szt") are not mass-convertible.
var gramsPerUnit = map[string]float64{
	Gram:       1,
	Kilogram:   1000,
	Milligram:  0.001,
	Milliliter: 1,
	Liter:      1000,
}

// Grams converts amount of unit into grams. The bool is false when the unit has
// no mass equivalent (e.g. "szt"), so callers can skip its macro contribution.
func Grams(amount float64, unit string) (float64, bool) {
	factor, ok := gramsPerUnit[normalize(unit)]
	if !ok {
		return 0, false
	}
	return amount * factor, true
}

// Known reports whether unit is one the app recognizes.
func Known(unit string) bool {
	switch normalize(unit) {
	case Gram, Kilogram, Milligram, Milliliter, Liter, Piece:
		return true
	default:
		return false
	}
}

func normalize(unit string) string {
	return strings.ToLower(strings.TrimSpace(unit))
}
