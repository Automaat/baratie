package dbutil

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	return hasPgCode(err, pgerrcode.UniqueViolation)
}

// IsForeignKeyViolation reports whether err is a Postgres foreign-key
// violation (SQLSTATE 23503).
func IsForeignKeyViolation(err error) bool {
	return hasPgCode(err, pgerrcode.ForeignKeyViolation)
}

func hasPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil {
		return false
	}
	return pgErr.Code == code
}
