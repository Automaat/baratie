package foods

import (
	"strconv"
	"strings"

	"github.com/Automaat/baratie/backend-go/internal/units"
)

// parsedLine is a best-effort interpretation of a free-form ingredient string.
type parsedLine struct {
	Name   string
	Amount float64
	Unit   string
}

// parseIngredientLine makes a best-effort split of a free-form ingredient line
// ("200 g chicken breast", "2 eggs", "salt") into a normalized food name,
// amount and unit. The name is lowercased so the migration dedupes common
// ingredients. An empty name (e.g. "200 g" with no food) signals "skip".
func parseIngredientLine(raw string) parsedLine {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return parsedLine{}
	}

	amount, unit, rest, ok := leadingQuantity(fields)
	if !ok {
		// No leading number: treat the whole line as a unitless food.
		return parsedLine{Name: normalizeName(raw), Amount: 0, Unit: units.Piece}
	}
	name := normalizeName(strings.Join(rest, " "))
	if name == "" {
		return parsedLine{}
	}
	return parsedLine{Name: name, Amount: amount, Unit: unit}
}

// leadingQuantity extracts a leading amount and (optional) unit from the token
// stream. It handles a separate unit token ("200 g x"), a glued unit ("200g x"),
// and a bare count ("2 eggs" → piece). The bool is false when the first token
// has no numeric prefix.
func leadingQuantity(fields []string) (float64, string, []string, bool) {
	num, suffix := splitLeadingNumber(fields[0])
	if num == "" {
		return 0, "", nil, false
	}
	amount, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, "", nil, false
	}
	if suffix != "" {
		if units.Known(suffix) {
			return amount, strings.ToLower(suffix), fields[1:], true
		}
		// Glued non-unit suffix ("2x"): keep it as part of the name.
		return amount, units.Piece, append([]string{suffix}, fields[1:]...), true
	}
	if len(fields) > 1 && units.Known(fields[1]) {
		return amount, strings.ToLower(fields[1]), fields[2:], true
	}
	return amount, units.Piece, fields[1:], true
}

// splitLeadingNumber splits a token into its leading numeric part and the rest
// ("200g" → "200","g"; "200" → "200",""; "abc" → "","abc").
func splitLeadingNumber(tok string) (string, string) {
	i := 0
	for i < len(tok) && (tok[i] >= '0' && tok[i] <= '9' || tok[i] == '.' || tok[i] == ',') {
		i++
	}
	return strings.ReplaceAll(tok[:i], ",", "."), tok[i:]
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
