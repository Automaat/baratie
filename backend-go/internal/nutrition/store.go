// Package nutrition implements the /api/nutrition/summary endpoint: macro
// totals aggregated from the meal plan over a date range. The store returns one
// row per planned meal (its recipe's per-serving macros); the grouping into
// per-day and period totals is done in pure Go (see summary.go) so the
// aggregation is unit-testable without a database.
package nutrition

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
)

// Contribution is one planned meal's macro contribution on a given date. A
// note-only meal (or one whose recipe was deleted) contributes zero macros via
// the LEFT JOIN + COALESCE, but still counts as a planned meal for that day.
type Contribution struct {
	Date         time.Time
	CaloriesKcal float64
	ProteinG     float64
	CarbsG       float64
	FatG         float64
}

// Store is the persistence boundary for nutrition aggregation.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Contributions returns one row per planned meal within the optional [from, to]
// date range (nil bounds are open), each carrying the linked recipe's
// per-serving macros, ordered by date then insertion order.
func (s *Store) Contributions(ctx context.Context, from, to *time.Time) ([]Contribution, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.plan_date,
		       COALESCE(r.calories_kcal, 0),
		       COALESCE(r.protein_g, 0),
		       COALESCE(r.carbs_g, 0),
		       COALESCE(r.fat_g, 0)
		FROM meal_plan_entries m
		LEFT JOIN recipes r ON r.id = m.recipe_id
		WHERE ($1::date IS NULL OR m.plan_date >= $1)
		  AND ($2::date IS NULL OR m.plan_date <= $2)
		ORDER BY m.plan_date, m.id`, from, to)
	if err != nil {
		return nil, fmt.Errorf("select nutrition contributions: %w", err)
	}
	return dbutil.CollectRows(rows, scanContribution,
		"scan nutrition contribution", "iterate nutrition contributions")
}

func scanContribution(row pgx.Row) (Contribution, error) {
	var c Contribution
	if err := row.Scan(&c.Date, &c.CaloriesKcal, &c.ProteinG, &c.CarbsG, &c.FatG); err != nil {
		return Contribution{}, err
	}
	return c, nil
}
