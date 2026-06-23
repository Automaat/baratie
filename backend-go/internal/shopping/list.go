package shopping

import (
	"sort"
	"strings"
)

// Item is one consolidated shopping-list line. Structured ingredients carry a
// summed Amount + Unit; free-form ingredients leave Amount 0 and Unit empty.
type Item struct {
	Name     string
	Amount   float64
	Unit     string
	Recipes  []string
	InPantry bool
}

// aggregate is the per-ingredient accumulator.
type aggregate struct {
	name    string              // first-seen display form
	amount  float64             // summed amount (structured only)
	unit    string              // unit (structured only)
	recipes map[string]struct{} // source recipe names (set)
}

// build consolidates the planned recipes' ingredients. Structured ingredients
// are summed per food+unit; free-form ingredients (recipes without structured
// data) are deduped by normalized text. Both record their source recipes and a
// best-effort pantry cross-off.
func build(structured []StructuredLine, freeform []PlannedRecipe, pantryNames []string) []Item {
	pantry := normalizeNames(pantryNames)
	items := buildStructured(structured, pantry)
	items = append(items, buildFreeform(freeform, pantry)...)
	// Keep one stable, globally-sorted list (by name, then unit) regardless of
	// the structured/free-form split.
	sort.Slice(items, func(i, j int) bool {
		ni, nj := strings.ToLower(items[i].Name), strings.ToLower(items[j].Name)
		if ni != nj {
			return ni < nj
		}
		return items[i].Unit < items[j].Unit
	})
	return items
}

// buildStructured sums structured amounts per (food, unit).
func buildStructured(lines []StructuredLine, pantry []string) []Item {
	byKey := map[string]aggregate{}
	keys := []string{}
	for _, l := range lines {
		food := strings.TrimSpace(l.Food)
		if food == "" {
			continue
		}
		unit := strings.ToLower(strings.TrimSpace(l.Unit))
		key := strings.ToLower(food) + "\x00" + unit
		a, ok := byKey[key]
		if !ok {
			a = aggregate{name: food, unit: unit, recipes: map[string]struct{}{}}
			keys = append(keys, key)
		}
		a.amount += l.Amount
		if r := strings.TrimSpace(l.Recipe); r != "" {
			a.recipes[r] = struct{}{}
		}
		byKey[key] = a // amount is a value field, so write the struct back
	}
	sort.Strings(keys)

	items := make([]Item, 0, len(keys))
	for _, k := range keys {
		a := byKey[k]
		items = append(items, Item{
			Name:     a.name,
			Amount:   a.amount,
			Unit:     a.unit,
			Recipes:  sortedKeys(a.recipes),
			InPantry: inPantry(strings.ToLower(a.name), pantry),
		})
	}
	return items
}

// buildFreeform dedupes free-form ingredient strings by normalized text.
func buildFreeform(recipes []PlannedRecipe, pantry []string) []Item {
	byKey := map[string]aggregate{}
	keys := []string{}
	for _, r := range recipes {
		for _, raw := range r.Ingredients {
			display := strings.TrimSpace(raw)
			if display == "" {
				continue
			}
			key := strings.ToLower(display)
			a, ok := byKey[key]
			if !ok {
				a = aggregate{name: display, recipes: map[string]struct{}{}}
				byKey[key] = a
				keys = append(keys, key)
			}
			// a.recipes is a reference; mutating it needs no write-back.
			if name := strings.TrimSpace(r.Name); name != "" {
				a.recipes[name] = struct{}{}
			}
		}
	}
	sort.Strings(keys)

	items := make([]Item, 0, len(keys))
	for _, k := range keys {
		a := byKey[k]
		items = append(items, Item{
			Name:     a.name,
			Recipes:  sortedKeys(a.recipes),
			InPantry: inPantry(k, pantry),
		})
	}
	return items
}

// normalizeNames lowercases and trims the names, dropping blanks.
func normalizeNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if t := strings.ToLower(strings.TrimSpace(n)); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// inPantry reports whether the normalized ingredient text contains any pantry
// item name as a substring.
func inPantry(ingredientKey string, pantry []string) bool {
	for _, p := range pantry {
		if strings.Contains(ingredientKey, p) {
			return true
		}
	}
	return false
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
