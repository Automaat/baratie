// Package pantry implements the /api/pantry endpoints — items currently in
// stock. PUT is a full replace; DELETE is a hard delete.
package pantry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
)

// Item is a single pantry stock line.
type Item struct {
	ID        int
	Name      string
	Quantity  float64
	Unit      string
	Category  string
	ExpiresOn *time.Time
	CreatedAt time.Time
}

// ErrNotFound is returned when no row matches the supplied id.
var ErrNotFound = errors.New("pantry item not found")

// Store is the persistence boundary for pantry items.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `id, name, quantity, unit, category, expires_on, created_at`

// List returns every pantry item ordered by category then name.
func (s *Store) List(ctx context.Context) ([]Item, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+selectColumns+` FROM pantry_items ORDER BY category, name`)
	if err != nil {
		return nil, fmt.Errorf("select pantry items: %w", err)
	}
	return dbutil.CollectRows(rows, scanItem, "scan pantry item", "iterate pantry items")
}

// Get returns a pantry item by id; ErrNotFound when absent.
func (s *Store) Get(ctx context.Context, id int) (*Item, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+selectColumns+` FROM pantry_items WHERE id = $1`, id)
	it, err := scanItem(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "select pantry item")
	}
	return &it, nil
}

// Create inserts a new pantry item.
func (s *Store) Create(ctx context.Context, it *Item) (*Item, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO pantry_items (name, quantity, unit, category, expires_on, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+selectColumns,
		it.Name, it.Quantity, it.Unit, it.Category, it.ExpiresOn, time.Now().UTC(),
	)
	created, err := scanItem(row)
	if err != nil {
		return nil, fmt.Errorf("insert pantry item: %w", err)
	}
	return &created, nil
}

// Update replaces every editable field; ErrNotFound if the id is unknown.
func (s *Store) Update(ctx context.Context, id int, it *Item) (*Item, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE pantry_items SET
			name = $1, quantity = $2, unit = $3, category = $4, expires_on = $5
		WHERE id = $6
		RETURNING `+selectColumns,
		it.Name, it.Quantity, it.Unit, it.Category, it.ExpiresOn, id,
	)
	updated, err := scanItem(row)
	if err != nil {
		return nil, dbutil.MapErr(err, ErrNotFound, "update pantry item")
	}
	return &updated, nil
}

// Delete removes the pantry item (hard delete); ErrNotFound when no row matched.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM pantry_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete pantry item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	if err := row.Scan(
		&it.ID, &it.Name, &it.Quantity, &it.Unit, &it.Category, &it.ExpiresOn, &it.CreatedAt,
	); err != nil {
		return Item{}, err
	}
	return it, nil
}
