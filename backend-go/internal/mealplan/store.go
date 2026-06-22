// Package mealplan implements the /api/meal-plan endpoints — dated meal
// entries that optionally reference a recipe. PUT is a full replace; DELETE is
// a hard delete.
package mealplan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
)

// Entry is a single planned meal. RecipeName is the joined recipe title (nil
// when no recipe is linked or the recipe was deleted).
type Entry struct {
	ID         int
	PlanDate   time.Time
	MealType   string
	RecipeID   *int
	RecipeName *string
	Note       string
	CreatedAt  time.Time
}

// ErrNotFound is returned when no row matches the supplied id.
var ErrNotFound = errors.New("meal plan entry not found")

// RecipeMissingError is returned when create/update references a non-existent
// recipe_id; mapped to a 404.
type RecipeMissingError struct {
	RecipeID int
}

func (e *RecipeMissingError) Error() string {
	return fmt.Sprintf("recipe with id %d not found", e.RecipeID)
}

// Store is the persistence boundary for meal plan entries.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `
	m.id, m.plan_date, m.meal_type, m.recipe_id, r.name, m.note, m.created_at
`

const fromJoin = ` FROM meal_plan_entries m LEFT JOIN recipes r ON r.id = m.recipe_id `

// List returns entries within the optional [from, to] date range (nil bounds
// are open), ordered by date then insertion order.
func (s *Store) List(ctx context.Context, from, to *time.Time) ([]Entry, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+selectColumns+fromJoin+`
		WHERE ($1::date IS NULL OR m.plan_date >= $1)
		  AND ($2::date IS NULL OR m.plan_date <= $2)
		ORDER BY m.plan_date, m.id`, from, to)
	if err != nil {
		return nil, fmt.Errorf("select meal plan: %w", err)
	}
	return dbutil.CollectRows(rows, scanEntry, "scan meal plan entry", "iterate meal plan")
}

// Get returns a single entry by id; ErrNotFound when absent.
func (s *Store) Get(ctx context.Context, id int) (*Entry, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+selectColumns+fromJoin+` WHERE m.id = $1`, id)
	e, err := scanEntry(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "select meal plan entry")
	}
	return &e, nil
}

// Create inserts a new entry. Returns RecipeMissingError if recipe_id is set
// and the referenced recipe doesn't exist.
func (s *Store) Create(ctx context.Context, e *Entry) (*Entry, error) {
	if err := s.validateRecipe(ctx, e.RecipeID); err != nil {
		return nil, err
	}
	var id int
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO meal_plan_entries (plan_date, meal_type, recipe_id, note, created_at)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		e.PlanDate, e.MealType, e.RecipeID, e.Note, time.Now().UTC(),
	).Scan(&id); err != nil {
		return nil, fmt.Errorf("insert meal plan entry: %w", err)
	}
	return s.Get(ctx, id)
}

// Update replaces every editable field; ErrNotFound if the id is unknown,
// RecipeMissingError if the new recipe_id doesn't exist.
func (s *Store) Update(ctx context.Context, id int, e *Entry) (*Entry, error) {
	if err := s.validateRecipe(ctx, e.RecipeID); err != nil {
		return nil, err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE meal_plan_entries SET
			plan_date = $1, meal_type = $2, recipe_id = $3, note = $4
		WHERE id = $5`,
		e.PlanDate, e.MealType, e.RecipeID, e.Note, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update meal plan entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Delete removes the entry (hard delete); ErrNotFound when no row matched.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM meal_plan_entries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete meal plan entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) validateRecipe(ctx context.Context, recipeID *int) error {
	if recipeID == nil {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM recipes WHERE id = $1)`, *recipeID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check recipe: %w", err)
	}
	if !exists {
		return &RecipeMissingError{RecipeID: *recipeID}
	}
	return nil
}

func scanEntry(row pgx.Row) (Entry, error) {
	var e Entry
	if err := row.Scan(
		&e.ID, &e.PlanDate, &e.MealType, &e.RecipeID, &e.RecipeName, &e.Note, &e.CreatedAt,
	); err != nil {
		return Entry{}, err
	}
	return e, nil
}
