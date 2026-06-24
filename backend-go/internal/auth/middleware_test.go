package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTokenFromRequestPrefersHeader pins the precedence the PAT flow relies on:
// an explicit Authorization header wins over an ambient session cookie, so a
// revoked bearer is actually evaluated rather than masked by a cookie that
// happens to ride along.
func TestTokenFromRequestPrefersHeader(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/me", http.NoBody)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: "cookie-jwt"})
	r.Header.Set("Authorization", "Bearer header-token")
	if got := tokenFromRequest(r); got != "header-token" {
		t.Fatalf("tokenFromRequest = %q, want Authorization header to win", got)
	}
}

// TestTokenFromRequestFallsBackToCookie covers the browser path: no header, the
// brt_token cookie is used.
func TestTokenFromRequestFallsBackToCookie(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/me", http.NoBody)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: "cookie-jwt"})
	if got := tokenFromRequest(r); got != "cookie-jwt" {
		t.Fatalf("tokenFromRequest = %q, want cookie fallback", got)
	}
}
