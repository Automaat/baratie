// Package recipes implements the /api/recipes endpoints.
//
// PUT is a full replace (every editable field is sent on each update); DELETE
// is a hard delete. Ingredients and tags are stored as Postgres text[] arrays.
package recipes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
)

// Recipe is a single recipe with its ingredient lines and free-form tags.
// The macro fields (CaloriesKcal, ProteinG, CarbsG, FatG) are per serving.
type Recipe struct {
	ID           int
	Name         string
	Description  string
	Instructions string
	Ingredients  []string
	Tags         []string
	Servings     int
	PrepMinutes  int
	CookMinutes  int
	CaloriesKcal float64
	ProteinG     float64
	CarbsG       float64
	FatG         float64
	CreatedAt    time.Time
}

// ErrNotFound is returned when no row matches the supplied id.
var ErrNotFound = errors.New("recipe not found")

// Store is the persistence boundary for recipes.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// EnsureSchema applies additive migrations to the recipes table for existing
// databases. The baseline schema.sql runs only on empty databases (see
// db.ApplySchema), so column additions must also run here as idempotent DDL.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		ALTER TABLE recipes
			ADD COLUMN IF NOT EXISTS calories_kcal double precision NOT NULL DEFAULT 0,
			ADD COLUMN IF NOT EXISTS protein_g double precision NOT NULL DEFAULT 0,
			ADD COLUMN IF NOT EXISTS carbs_g double precision NOT NULL DEFAULT 0,
			ADD COLUMN IF NOT EXISTS fat_g double precision NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("ensure recipes nutrition columns: %w", err)
	}
	return nil
}

const selectColumns = `
	id, name, description, instructions, ingredients, tags,
	servings, prep_minutes, cook_minutes,
	calories_kcal, protein_g, carbs_g, fat_g, created_at
`

// List returns every recipe ordered by name.
func (s *Store) List(ctx context.Context) ([]Recipe, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+selectColumns+` FROM recipes ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("select recipes: %w", err)
	}
	return dbutil.CollectRows(rows, scanRecipe, "scan recipe", "iterate recipes")
}

// Get returns a recipe by id; ErrNotFound when absent.
func (s *Store) Get(ctx context.Context, id int) (*Recipe, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+selectColumns+` FROM recipes WHERE id = $1`, id)
	r, err := scanRecipe(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "select recipe")
	}
	return &r, nil
}

// Create inserts a new recipe and returns the stored row.
func (s *Store) Create(ctx context.Context, r *Recipe) (*Recipe, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO recipes (
			name, description, instructions, ingredients, tags,
			servings, prep_minutes, cook_minutes,
			calories_kcal, protein_g, carbs_g, fat_g, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING `+selectColumns,
		r.Name, r.Description, r.Instructions, r.Ingredients, r.Tags,
		r.Servings, r.PrepMinutes, r.CookMinutes,
		r.CaloriesKcal, r.ProteinG, r.CarbsG, r.FatG, time.Now().UTC(),
	)
	created, err := scanRecipe(row)
	if err != nil {
		return nil, fmt.Errorf("insert recipe: %w", err)
	}
	return &created, nil
}

// Update replaces every editable field of a recipe; ErrNotFound if the id is
// unknown.
func (s *Store) Update(ctx context.Context, id int, r *Recipe) (*Recipe, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE recipes SET
			name = $1, description = $2, instructions = $3, ingredients = $4,
			tags = $5, servings = $6, prep_minutes = $7, cook_minutes = $8,
			calories_kcal = $9, protein_g = $10, carbs_g = $11, fat_g = $12
		WHERE id = $13
		RETURNING `+selectColumns,
		r.Name, r.Description, r.Instructions, r.Ingredients, r.Tags,
		r.Servings, r.PrepMinutes, r.CookMinutes,
		r.CaloriesKcal, r.ProteinG, r.CarbsG, r.FatG, id,
	)
	updated, err := scanRecipe(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "update recipe")
	}
	return &updated, nil
}

// Delete removes the recipe (hard delete); ErrNotFound when no row matched.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete recipe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanRecipe(row pgx.Row) (Recipe, error) {
	var r Recipe
	if err := row.Scan(
		&r.ID, &r.Name, &r.Description, &r.Instructions, &r.Ingredients, &r.Tags,
		&r.Servings, &r.PrepMinutes, &r.CookMinutes,
		&r.CaloriesKcal, &r.ProteinG, &r.CarbsG, &r.FatG, &r.CreatedAt,
	); err != nil {
		return Recipe{}, err
	}
	return r, nil
}
