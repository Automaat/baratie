package foods

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
)

// recipeLines pairs a recipe id with its free-form ingredient strings.
type recipeLines struct {
	id          int
	ingredients []string
}

// MigrateFreeformIngredients does a one-time, idempotent best-effort import of
// free-form recipe ingredient strings into the structured foods +
// recipe_ingredients model. It only touches recipes that have free-form
// ingredients and no structured ingredients yet, never deletes the originals,
// and creates foods with zero macros (filled in later via the UI). Each recipe
// is migrated in its own transaction; a single recipe's failure is logged and
// skipped rather than aborting startup. Returns the number of recipes migrated.
func (s *Store) MigrateFreeformIngredients(ctx context.Context, logger *slog.Logger) (int, error) {
	if logger == nil {
		logger = slog.Default()
	}
	pending, err := s.unmigratedRecipes(ctx)
	if err != nil {
		return 0, err
	}
	migrated := 0
	for _, r := range pending {
		if err := s.migrateRecipe(ctx, r); err != nil {
			logger.Error("migrate recipe ingredients", "recipe_id", r.id, "err", err)
			continue
		}
		migrated++
	}
	return migrated, nil
}

// unmigratedRecipes returns recipes that have free-form ingredients but no
// structured ones yet.
func (s *Store) unmigratedRecipes(ctx context.Context) ([]recipeLines, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.ingredients
		FROM recipes r
		WHERE cardinality(r.ingredients) > 0
		  AND NOT EXISTS (SELECT 1 FROM recipe_ingredients ri WHERE ri.recipe_id = r.id)
		ORDER BY r.id`)
	if err != nil {
		return nil, fmt.Errorf("select unmigrated recipes: %w", err)
	}
	return dbutil.CollectRows(rows, func(row pgx.Row) (recipeLines, error) {
		var r recipeLines
		if err := row.Scan(&r.id, &r.ingredients); err != nil {
			return recipeLines{}, err
		}
		return r, nil
	}, "scan unmigrated recipe", "iterate unmigrated recipes")
}

// migrateRecipe imports one recipe's ingredient lines in a single transaction.
func (s *Store) migrateRecipe(ctx context.Context, r recipeLines) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migrate tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for pos, line := range r.ingredients {
		pl := parseIngredientLine(line)
		if pl.Name == "" {
			continue
		}
		foodID, err := upsertFoodTx(ctx, tx, pl.Name)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, food_id, amount, unit, position)
			VALUES ($1, $2, $3, $4, $5)`,
			r.id, foodID, pl.Amount, pl.Unit, pos); err != nil {
			return fmt.Errorf("insert migrated ingredient: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrate tx: %w", err)
	}
	return nil
}

// upsertFoodTx returns the id of the food with the given name, creating it
// (with zero macros) if absent. The ON CONFLICT DO UPDATE returns the id on
// both insert and conflict.
func upsertFoodTx(ctx context.Context, tx pgx.Tx, name string) (int, error) {
	var id int
	if err := tx.QueryRow(ctx, `
		INSERT INTO foods (name) VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, name).Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert food %q: %w", name, err)
	}
	return id, nil
}
