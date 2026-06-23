// Package foods implements the /api/foods food library (per-100g macros) plus
// the schema and best-effort migration for structured recipe ingredients
// (issue #5). The recipe_ingredients junction is owned here because it depends
// on the foods catalog; the recipes package reads/writes the links.
package foods

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
)

// Food is one entry in the food library, with macros per 100 g.
type Food struct {
	ID             int
	Name           string
	KcalPer100g    float64
	ProteinPer100g float64
	CarbsPer100g   float64
	FatPer100g     float64
	CreatedAt      time.Time
}

// Sentinel errors so handlers map to HTTP status without sniffing pg text.
var (
	ErrNotFound     = errors.New("food not found")
	ErrNameConflict = errors.New("food name already exists")
	ErrInUse        = errors.New("food is used by a recipe")
)

// Store is the persistence boundary for the food library.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// EnsureSchema creates the foods and recipe_ingredients tables if absent. The
// baseline schema.sql runs only on empty databases (see db.ApplySchema), so
// this additive DDL also runs against existing databases and must be
// idempotent. It must run after the recipes table exists (FK target).
func (s *Store) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS foods (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name varchar(200) NOT NULL UNIQUE,
			kcal_per_100g double precision NOT NULL DEFAULT 0,
			protein_per_100g double precision NOT NULL DEFAULT 0,
			carbs_per_100g double precision NOT NULL DEFAULT 0,
			fat_per_100g double precision NOT NULL DEFAULT 0,
			created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
		)`,
		`CREATE TABLE IF NOT EXISTS recipe_ingredients (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			recipe_id integer NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
			food_id integer NOT NULL REFERENCES foods (id) ON DELETE RESTRICT,
			amount double precision NOT NULL DEFAULT 0,
			unit varchar(50) NOT NULL DEFAULT 'g',
			position integer NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_recipe ON recipe_ingredients (recipe_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("ensure foods schema: %w", err)
		}
	}
	return nil
}

const foodColumns = `id, name, kcal_per_100g, protein_per_100g, carbs_per_100g, fat_per_100g, created_at`

// List returns every food ordered by name.
func (s *Store) List(ctx context.Context) ([]Food, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+foodColumns+` FROM foods ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("select foods: %w", err)
	}
	return dbutil.CollectRows(rows, scanFood, "scan food", "iterate foods")
}

// Get returns a food by id; ErrNotFound when absent.
func (s *Store) Get(ctx context.Context, id int) (*Food, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+foodColumns+` FROM foods WHERE id = $1`, id)
	f, err := scanFood(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "select food")
	}
	return &f, nil
}

// Create inserts a new food; ErrNameConflict on duplicate name.
func (s *Store) Create(ctx context.Context, f *Food) (*Food, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO foods (name, kcal_per_100g, protein_per_100g, carbs_per_100g, fat_per_100g, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+foodColumns,
		f.Name, f.KcalPer100g, f.ProteinPer100g, f.CarbsPer100g, f.FatPer100g, time.Now().UTC(),
	)
	created, err := scanFood(row)
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("insert food: %w", err)
	}
	return &created, nil
}

// Update replaces every editable field of a food; ErrNotFound if the id is
// unknown, ErrNameConflict on a duplicate name.
func (s *Store) Update(ctx context.Context, id int, f *Food) (*Food, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE foods SET
			name = $1, kcal_per_100g = $2, protein_per_100g = $3,
			carbs_per_100g = $4, fat_per_100g = $5
		WHERE id = $6
		RETURNING `+foodColumns,
		f.Name, f.KcalPer100g, f.ProteinPer100g, f.CarbsPer100g, f.FatPer100g, id,
	)
	updated, err := scanFood(row)
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, ErrNameConflict
		}
		return nil, dbutil.MapErr(err, ErrNotFound, "update food")
	}
	return &updated, nil
}

// Delete removes a food (hard delete); ErrNotFound when no row matched,
// ErrInUse when a recipe still references it.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM foods WHERE id = $1`, id)
	if err != nil {
		if dbutil.IsForeignKeyViolation(err) {
			return ErrInUse
		}
		return fmt.Errorf("delete food: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanFood(row pgx.Row) (Food, error) {
	var f Food
	if err := row.Scan(&f.ID, &f.Name, &f.KcalPer100g, &f.ProteinPer100g,
		&f.CarbsPer100g, &f.FatPer100g, &f.CreatedAt); err != nil {
		return Food{}, err
	}
	return f, nil
}
