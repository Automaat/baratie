package recipes

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
	"github.com/Automaat/baratie/backend-go/internal/units"
)

// ErrFoodMissing is returned when a structured ingredient references a food id
// that does not exist; mapped to a 422.
var ErrFoodMissing = errors.New("food not found")

// StructuredIngredient is one recipe→food link, joined to the food's per-100g
// macros, as returned on reads.
type StructuredIngredient struct {
	ID             int
	FoodID         int
	FoodName       string
	Amount         float64
	Unit           string
	Position       int
	KcalPer100g    float64
	ProteinPer100g float64
	CarbsPer100g   float64
	FatPer100g     float64
}

// IngredientInput is one structured ingredient as accepted on writes.
type IngredientInput struct {
	FoodID int
	Amount float64
	Unit   string
}

// computedMacros is a total/per-serving macro set derived from ingredients.
type computedMacros struct {
	Kcal    float64
	Protein float64
	Carbs   float64
	Fat     float64
}

// querier is the subset of pgxpool.Pool / pgx.Tx the ingredient reads need.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

const ingredientColumns = `
	ri.id, ri.food_id, f.name, ri.amount, ri.unit, ri.position,
	f.kcal_per_100g, f.protein_per_100g, f.carbs_per_100g, f.fat_per_100g
`

// IngredientsByRecipe returns a recipe's structured ingredients, ordered.
func (s *Store) IngredientsByRecipe(ctx context.Context, recipeID int) ([]StructuredIngredient, error) {
	return selectIngredients(ctx, s.pool, recipeID)
}

// IngredientsForRecipes returns structured ingredients grouped by recipe id for
// the given recipes (one query, grouped in Go to avoid N+1).
func (s *Store) IngredientsForRecipes(ctx context.Context, ids []int) (map[int][]StructuredIngredient, error) {
	out := map[int][]StructuredIngredient{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT ri.recipe_id, `+ingredientColumns+`
		FROM recipe_ingredients ri JOIN foods f ON f.id = ri.food_id
		WHERE ri.recipe_id = ANY($1)
		ORDER BY ri.recipe_id, ri.position, ri.id`, ids)
	if err != nil {
		return nil, fmt.Errorf("select recipe ingredients: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rid int
		si, err := scanWithRecipeID(rows, &rid)
		if err != nil {
			return nil, fmt.Errorf("scan recipe ingredient: %w", err)
		}
		out[rid] = append(out[rid], si)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipe ingredients: %w", err)
	}
	return out, nil
}

// ReplaceIngredients full-replaces a recipe's structured ingredients in one
// transaction, then recomputes and persists the recipe's per-serving macro
// columns when the linked foods carry usable macro data (so reads stay a single
// source of truth). ErrNotFound if the recipe is unknown, ErrFoodMissing if an
// input references a missing food. Returns the refreshed recipe.
func (s *Store) ReplaceIngredients(ctx context.Context, recipeID int, inputs []IngredientInput) (*Recipe, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin ingredients tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var servings int
	if err := tx.QueryRow(ctx, `SELECT servings FROM recipes WHERE id = $1`, recipeID).Scan(&servings); err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "select recipe servings")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM recipe_ingredients WHERE recipe_id = $1`, recipeID); err != nil {
		return nil, fmt.Errorf("clear recipe ingredients: %w", err)
	}
	for pos, in := range inputs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, food_id, amount, unit, position)
			VALUES ($1, $2, $3, $4, $5)`,
			recipeID, in.FoodID, in.Amount, in.Unit, pos); err != nil {
			if dbutil.IsForeignKeyViolation(err) {
				return nil, ErrFoodMissing
			}
			return nil, fmt.Errorf("insert recipe ingredient: %w", err)
		}
	}
	if err := recomputeMacros(ctx, tx, recipeID, servings); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ingredients tx: %w", err)
	}
	return s.Get(ctx, recipeID)
}

// recomputeMacros reads the recipe's structured ingredients and, when they
// carry usable macro data, overwrites the recipe's per-serving macro columns
// with the computed values.
func recomputeMacros(ctx context.Context, tx pgx.Tx, recipeID, servings int) error {
	ings, err := selectIngredients(ctx, tx, recipeID)
	if err != nil {
		return err
	}
	cm, usable := computePerServing(ings, servings)
	if !usable {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE recipes SET calories_kcal = $1, protein_g = $2, carbs_g = $3, fat_g = $4
		WHERE id = $5`, cm.Kcal, cm.Protein, cm.Carbs, cm.Fat, recipeID); err != nil {
		return fmt.Errorf("persist computed macros: %w", err)
	}
	return nil
}

// computePerServing sums the ingredients' macros (converting amounts to grams
// against per-100g food data) and divides by servings. usable is false when no
// ingredient yields any macro value (e.g. all zero-macro or non-mass units), so
// callers can keep the existing manual macros instead.
func computePerServing(ings []StructuredIngredient, servings int) (computedMacros, bool) {
	if servings < 1 {
		servings = 1
	}
	var total computedMacros
	for _, ing := range ings {
		grams, ok := units.Grams(ing.Amount, ing.Unit)
		if !ok {
			continue
		}
		f := grams / 100.0
		total.Kcal += ing.KcalPer100g * f
		total.Protein += ing.ProteinPer100g * f
		total.Carbs += ing.CarbsPer100g * f
		total.Fat += ing.FatPer100g * f
	}
	usable := total.Kcal > 0 || total.Protein > 0 || total.Carbs > 0 || total.Fat > 0
	per := computedMacros{
		Kcal:    total.Kcal / float64(servings),
		Protein: total.Protein / float64(servings),
		Carbs:   total.Carbs / float64(servings),
		Fat:     total.Fat / float64(servings),
	}
	return per, usable
}

func selectIngredients(ctx context.Context, q querier, recipeID int) ([]StructuredIngredient, error) {
	rows, err := q.Query(ctx, `SELECT `+ingredientColumns+`
		FROM recipe_ingredients ri JOIN foods f ON f.id = ri.food_id
		WHERE ri.recipe_id = $1 ORDER BY ri.position, ri.id`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("select recipe ingredients: %w", err)
	}
	return dbutil.CollectRows(rows, scanIngredient, "scan recipe ingredient", "iterate recipe ingredients")
}

func scanIngredient(row pgx.Row) (StructuredIngredient, error) {
	var si StructuredIngredient
	if err := row.Scan(&si.ID, &si.FoodID, &si.FoodName, &si.Amount, &si.Unit, &si.Position,
		&si.KcalPer100g, &si.ProteinPer100g, &si.CarbsPer100g, &si.FatPer100g); err != nil {
		return StructuredIngredient{}, err
	}
	return si, nil
}

func scanWithRecipeID(row pgx.Row, recipeID *int) (StructuredIngredient, error) {
	var si StructuredIngredient
	if err := row.Scan(recipeID, &si.ID, &si.FoodID, &si.FoodName, &si.Amount, &si.Unit, &si.Position,
		&si.KcalPer100g, &si.ProteinPer100g, &si.CarbsPer100g, &si.FatPer100g); err != nil {
		return StructuredIngredient{}, err
	}
	return si, nil
}
