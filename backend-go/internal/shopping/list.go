package shopping

import (
	"sort"
	"strings"
)

// Item is one consolidated shopping-list line: a deduped ingredient, the
// recipes that call for it, and whether it appears to be in the pantry already.
type Item struct {
	Name     string
	Recipes  []string
	InPantry bool
}

// aggregate is the per-ingredient accumulator keyed by normalized text.
type aggregate struct {
	name    string              // first-seen display form (trimmed original)
	recipes map[string]struct{} // source recipe names (set)
}

// build consolidates the ingredient lines of the planned recipes into a deduped
// list (by case-insensitive trimmed text), records which recipes call for each
// ingredient, and flags items whose text contains a pantry item name
// (best-effort cross-off — free-form strings limit accuracy).
func build(recipes []PlannedRecipe, pantryNames []string) []Item {
	pantry := normalizeNames(pantryNames)

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
