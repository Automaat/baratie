// Package shopping implements the /api/shopping-list endpoint: a consolidated
// ingredient list collected from the recipes planned over a date range. With
// free-form ingredient strings the list is deduped by normalized text (true
// quantity summing needs structured ingredients — a separate upgrade). The
// store reads the raw planned recipes + pantry names; the consolidation is a
// pure Go function (see list.go) so it is unit-testable without a database.
package shopping

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
)

// PlannedRecipe is a recipe referenced by the meal plan in range, with its
// free-form ingredient lines.
type PlannedRecipe struct {
	Name        string
	Ingredients []string
}

// Store is the persistence boundary for shopping-list aggregation.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// PlannedRecipes returns the distinct recipes planned within the optional
// [from, to] date range (nil bounds are open), each with its ingredient lines,
// ordered by name. Meal-plan entries without a linked recipe are skipped.
func (s *Store) PlannedRecipes(ctx context.Context, from, to *time.Time) ([]PlannedRecipe, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT r.name, r.ingredients
		FROM meal_plan_entries m
		JOIN recipes r ON r.id = m.recipe_id
		WHERE ($1::date IS NULL OR m.plan_date >= $1)
		  AND ($2::date IS NULL OR m.plan_date <= $2)
		ORDER BY r.name`, from, to)
	if err != nil {
		return nil, fmt.Errorf("select planned recipes: %w", err)
	}
	return dbutil.CollectRows(rows, scanPlannedRecipe,
		"scan planned recipe", "iterate planned recipes")
}

// PantryNames returns the names of pantry items currently in stock
// (quantity > 0), used for best-effort cross-off of ingredients already on
// hand. Items at zero quantity are excluded so they aren't flagged as stocked.
func (s *Store) PantryNames(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT name FROM pantry_items WHERE quantity > 0`)
	if err != nil {
		return nil, fmt.Errorf("select pantry names: %w", err)
	}
	return dbutil.CollectRows(rows, scanName, "scan pantry name", "iterate pantry names")
}

func scanPlannedRecipe(row pgx.Row) (PlannedRecipe, error) {
	var p PlannedRecipe
	if err := row.Scan(&p.Name, &p.Ingredients); err != nil {
		return PlannedRecipe{}, err
	}
	return p, nil
}

func scanName(row pgx.Row) (string, error) {
	var name string
	if err := row.Scan(&name); err != nil {
		return "", err
	}
	return name, nil
}
