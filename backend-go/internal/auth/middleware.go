package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
)

// CookieName is the session cookie carrying the JWT.
const CookieName = "brt_token"

type ctxKey int

const claimsKey ctxKey = 0

// Authenticate accepts either a session JWT or a personal access token — taken
// from the brt_token cookie or a Bearer Authorization header — and stores the
// resolved claims in the request context. A token carrying the PAT prefix is
// looked up in the database (no 24h expiry); anything else is verified as a
// JWT. Requests without a valid token get 401.
func Authenticate(tokens *TokenService, pats *PATStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := tokenFromRequest(r)
			if raw == "" {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			claims, err := resolveClaims(r.Context(), tokens, pats, raw)
			if err != nil {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
		})
	}
}

// resolveClaims dispatches on the token shape: a PAT-prefixed bearer goes to
// the database-backed token store, everything else to the JWT verifier.
func resolveClaims(ctx context.Context, tokens *TokenService, pats *PATStore, raw string) (*Claims, error) {
	if pats != nil && strings.HasPrefix(raw, PATPrefix) {
		return pats.Authenticate(ctx, raw)
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
