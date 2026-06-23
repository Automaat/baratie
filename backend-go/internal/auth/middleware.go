package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
)

// CookieName is the session cookie carrying the JWT.
const CookieName = "brt_token"

type ctxKey int

const claimsKey ctxKey = 0

// errAuthUnavailable marks an authentication failure caused by an internal
// fault — the token store being unreachable — rather than a bad credential. A
// database blip while resolving a PAT must surface as 500, not a 401 "invalid
// token" that would tell a headless client to throw away a perfectly good
// token.
var errAuthUnavailable = errors.New("auth backend unavailable")

// Authenticate accepts either a session JWT or a personal access token — taken
// from the brt_token cookie or a Bearer Authorization header — and stores the
// resolved claims in the request context. A token carrying the PAT prefix is
// looked up in the database (no 24h expiry); anything else is verified as a
// JWT. A missing, bad or expired credential gets 401; a fault resolving it
// (e.g. the token DB is down) gets 500.
func Authenticate(tokens *TokenService, pats *PATStore, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := tokenFromRequest(r)
			if raw == "" {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			claims, err := resolveClaims(r.Context(), tokens, pats, raw)
			if err != nil {
				if errors.Is(err, errAuthUnavailable) {
					logger.Error("authenticate request", "err", err)
					httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
					return
				}
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
		})
	}
}

// resolveClaims dispatches on the token shape: a PAT-prefixed bearer goes to
// the database-backed token store, everything else to the JWT verifier. A PAT
// lookup that fails for any reason other than a missing/expired token is an
// infrastructure fault, tagged errAuthUnavailable so the middleware can tell it
// apart from a genuine bad credential. JWT verification is pure-CPU, so its
// errors are always bad credentials.
func resolveClaims(ctx context.Context, tokens *TokenService, pats *PATStore, raw string) (*Claims, error) {
	if pats != nil && strings.HasPrefix(raw, PATPrefix) {
		claims, err := pats.Authenticate(ctx, raw)
		if err != nil && !errors.Is(err, ErrTokenNotFound) {
			return nil, fmt.Errorf("%w: %w", errAuthUnavailable, err)
		}
		return claims, err
	}
	return tokens.Verify(raw)
}

// RequireAdmin rejects authenticated non-admin users with 403. It must run
// after Authenticate.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(r.Context())
		if !ok || !claims.IsAdmin {
			httputil.WriteDetailError(w, http.StatusForbidden, "Admin privileges required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	const prefix = "Bearer "
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, prefix) {
		return strings.TrimPrefix(h, prefix)
	}
	return ""
}

func claimsFrom(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}
