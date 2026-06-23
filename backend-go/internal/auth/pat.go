package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/dbutil"
)

// PATPrefix marks a personal access token so the middleware can tell it apart
// from a JWT (which is dot-separated base64url segments and never carries this
// prefix). The remainder is base64url-encoded random bytes.
const PATPrefix = "brt_pat_"

// patRandomBytes is the entropy behind a token (256 bits). High enough that a
// plain SHA-256 hash lookup — not a slow password hash — is safe.
const patRandomBytes = 32

// patScopeFull is the only scope today: a token grants its owner's full API
// access. The column is kept for forward compatibility (scoped tokens), but no
// scope enforcement exists yet.
const patScopeFull = "full"

// patLastUsedThrottle bounds how often last_used_at is rewritten. The stamp is
// an audit breadcrumb, not part of the auth decision, so a token used in a tight
// loop (a polling headless client) authenticates from a single read and the row
// is rewritten at most once per interval — keeping the hot path off the
// primary's write path and out of a per-row lock.
const patLastUsedThrottle = time.Minute

// ErrTokenNotFound is returned when a token id does not belong to the caller.
var ErrTokenNotFound = errors.New("token not found")

// PersonalAccessToken is the stored metadata for a long-lived API token. The
// secret itself is never retained — only its hash — so it is absent here.
type PersonalAccessToken struct {
	ID         int
	UserID     int
	Name       string
	Scope      string
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
}

// PATStore is the persistence boundary for personal access tokens.
type PATStore struct {
	pool *pgxpool.Pool
}

// NewPATStore wraps a pool.
func NewPATStore(pool *pgxpool.Pool) *PATStore {
	return &PATStore{pool: pool}
}

// EnsureSchema creates the personal_access_tokens table if absent. Like the
// other additive DDL it must be idempotent and run after the users table
// exists (FK target).
func (s *PATStore) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS personal_access_tokens (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			token_hash text NOT NULL UNIQUE,
			name varchar(100) NOT NULL,
			scope varchar(50) NOT NULL DEFAULT 'full',
			created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			expires_at timestamp without time zone,
			last_used_at timestamp without time zone
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pat_user ON personal_access_tokens (user_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("ensure personal_access_tokens schema: %w", err)
		}
	}
	return nil
}

const patColumns = `id, user_id, name, scope, created_at, expires_at, last_used_at`

// Create issues a new token for userID and stores only its hash. The returned
// plaintext token is shown to the caller once and is unrecoverable afterwards.
func (s *PATStore) Create(ctx context.Context, userID int, name string, expiresAt *time.Time) (string, *PersonalAccessToken, error) {
	raw, hash, err := generatePAT()
	if err != nil {
		return "", nil, err
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO personal_access_tokens (user_id, token_hash, name, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+patColumns,
		userID, hash, name, patScopeFull, expiresAt)
	pat, err := scanPAT(row)
	if err != nil {
		return "", nil, fmt.Errorf("insert token: %w", err)
	}
	return raw, &pat, nil
}

// List returns a user's tokens (metadata only), newest first.
func (s *PATStore) List(ctx context.Context, userID int) ([]PersonalAccessToken, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+patColumns+` FROM personal_access_tokens WHERE user_id = $1 ORDER BY created_at DESC, id DESC`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("select tokens: %w", err)
	}
	return dbutil.CollectRows(rows, scanPAT, "scan token", "iterate tokens")
}

// Delete revokes a token, scoped to its owner so one user cannot revoke
// another's. ErrTokenNotFound when no owned row matched.
func (s *PATStore) Delete(ctx context.Context, userID, id int) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM personal_access_tokens WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}
	return nil
}

// Authenticate validates a raw bearer token against the stored hashes. The
// lookup is a read: it matches the hash, rejects expired tokens and returns the
// owner's identity — yielding Claims with no JWT-style expiry. last_used_at is
// then refreshed best-effort and throttled (see touchLastUsed), so the hot path
// stays a single read and an already-authenticated request never fails on the
// breadcrumb write. ErrTokenNotFound when the token is unknown, revoked or
// expired.
func (s *PATStore) Authenticate(ctx context.Context, raw string) (*Claims, error) {
	if !strings.HasPrefix(raw, PATPrefix) {
		return nil, ErrTokenNotFound
	}
	row := s.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.name, u.is_admin, p.id, p.last_used_at
		FROM personal_access_tokens p
		JOIN users u ON p.user_id = u.id
		WHERE p.token_hash = $1
			AND (p.expires_at IS NULL OR p.expires_at > (now() AT TIME ZONE 'utc'))`,
		hashPAT(raw))
	var (
		uid        int
		username   string
		name       *string
		isAdmin    bool
		patID      int
		lastUsedAt *time.Time
	)
	if err := row.Scan(&uid, &username, &name, &isAdmin, &patID, &lastUsedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("authenticate token: %w", err)
	}
	s.touchLastUsed(ctx, patID, lastUsedAt)
	return &Claims{UserID: uid, Username: username, Name: derefName(name), IsAdmin: isAdmin}, nil
}

// touchLastUsed refreshes a token's last_used_at, but only once the previous
// stamp is older than patLastUsedThrottle, and never reports failure: the stamp
// is an audit breadcrumb on a request that is already authenticated, so a
// skipped or failed write must not turn into an auth error. Concurrent stale
// requests simply write the same value.
func (s *PATStore) touchLastUsed(ctx context.Context, id int, lastUsedAt *time.Time) {
	if !patStampDue(lastUsedAt, time.Now().UTC()) {
		return
	}
	// Best-effort: the auth decision is already made, so the result is dropped.
	_, _ = s.pool.Exec(ctx,
		`UPDATE personal_access_tokens SET last_used_at = (now() AT TIME ZONE 'utc') WHERE id = $1`,
		id)
}

// patStampDue reports whether last_used_at is stale enough to rewrite: never
// stamped, or last stamped at least patLastUsedThrottle ago.
func patStampDue(lastUsedAt *time.Time, now time.Time) bool {
	return lastUsedAt == nil || now.Sub(*lastUsedAt) >= patLastUsedThrottle
}

func scanPAT(row pgx.Row) (PersonalAccessToken, error) {
	var p PersonalAccessToken
	if err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Scope,
		&p.CreatedAt, &p.ExpiresAt, &p.LastUsedAt); err != nil {
		return PersonalAccessToken{}, err
	}
	return p, nil
}

// generatePAT returns a fresh plaintext token and its storage hash.
func generatePAT() (string, string, error) {
	buf := make([]byte, patRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	raw := PATPrefix + base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashPAT(raw), nil
}

// hashPAT is the at-rest representation of a token: a hex SHA-256 digest. The
// token's own entropy (256 bits) makes a fast hash sufficient and keeps lookup
// a single indexed equality.
func hashPAT(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
