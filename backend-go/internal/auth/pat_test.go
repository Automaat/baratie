package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestGeneratePATShape(t *testing.T) {
	raw, hash, err := generatePAT()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.HasPrefix(raw, PATPrefix) {
		t.Fatalf("token %q missing prefix %q", raw, PATPrefix)
	}
	if hash != hashPAT(raw) {
		t.Fatal("returned hash does not match hashPAT(raw)")
	}
	if len(hash) != 64 {
		t.Fatalf("hash len = %d, want 64 hex chars", len(hash))
	}
}

func TestGeneratePATUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		raw, _, err := generatePAT()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if _, dup := seen[raw]; dup {
			t.Fatalf("duplicate token generated: %q", raw)
		}
		seen[raw] = struct{}{}
	}
}

func TestHashPATDeterministic(t *testing.T) {
	const token = "brt_pat_abc"
	first := hashPAT(token)
	if first != hashPAT(token) {
		t.Fatal("hash not deterministic")
	}
	if first == hashPAT("brt_pat_abd") {
		t.Fatal("distinct tokens hashed to the same value")
	}
}

func TestValidateTokenName(t *testing.T) {
	if validateToken(&createTokenRequest{Name: "  "}) == nil {
		t.Fatal("blank name should fail")
	}
	if validateToken(&createTokenRequest{Name: strings.Repeat("x", 101)}) == nil {
		t.Fatal("over-long name should fail")
	}
	req := createTokenRequest{Name: "  coach  "}
	if vErr := validateToken(&req); vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if req.Name != "coach" {
		t.Fatalf("name = %q, want trimmed", req.Name)
	}
	if req.expiresAt != nil {
		t.Fatal("no expiry given, expiresAt should stay nil")
	}
}

func TestValidateTokenExpiry(t *testing.T) {
	bad := "not-a-date"
	if validateToken(&createTokenRequest{Name: "x", ExpiresAt: &bad}) == nil {
		t.Fatal("malformed expiry should fail")
	}
	past := "2000-01-01"
	if validateToken(&createTokenRequest{Name: "x", ExpiresAt: &past}) == nil {
		t.Fatal("past expiry should fail")
	}
	future := time.Now().UTC().Add(48 * time.Hour).Format("2006-01-02")
	req := createTokenRequest{Name: "x", ExpiresAt: &future}
	if vErr := validateToken(&req); vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if req.expiresAt == nil {
		t.Fatal("expiresAt not set")
	}
	// Pin the end-of-day semantics: a YYYY-MM-DD expiry must resolve to the
	// last microsecond of that UTC day, not its midnight. Guards the
	// Add(24h - time.Microsecond) in validateToken against a silent regression
	// that would expire tokens a full day early.
	day, _ := time.Parse("2006-01-02", future)
	wantExpiry := day.Add(24*time.Hour - time.Microsecond)
	if !req.expiresAt.Equal(wantExpiry) {
		t.Fatalf("expiresAt = %v, want end of day %v", req.expiresAt, wantExpiry)
	}
}

func TestPATStampDue(t *testing.T) {
	now := time.Now().UTC()
	if !patStampDue(nil, now) {
		t.Fatal("a never-stamped token should be due")
	}
	fresh := now.Add(-patLastUsedThrottle / 2)
	if patStampDue(&fresh, now) {
		t.Fatal("a token stamped within the throttle window should not be due")
	}
	stale := now.Add(-patLastUsedThrottle - time.Second)
	if !patStampDue(&stale, now) {
		t.Fatal("a token stamped past the throttle window should be due")
	}
}

// TestResolveClaimsJWTErrorIsCredential guards the 401-vs-500 split: a failed
// JWT verification is a bad credential (401), never an infrastructure fault, so
// it must not carry errAuthUnavailable.
func TestResolveClaimsJWTErrorIsCredential(t *testing.T) {
	ts := NewTokenService("test-secret")
	_, err := resolveClaims(context.Background(), ts, nil, "not-a-jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
	if errors.Is(err, errAuthUnavailable) {
		t.Fatal("JWT verify error must not be tagged as infra-unavailable")
	}
}
