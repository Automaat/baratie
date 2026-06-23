package auth

import (
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
	if req.expiresAt == nil || !req.expiresAt.After(time.Now().UTC()) {
		t.Fatalf("expiresAt not set to a future time: %v", req.expiresAt)
	}
}
