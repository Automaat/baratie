package auth

import (
	"testing"
	"time"
)

func TestTokenRoundTrip(t *testing.T) {
	ts := NewTokenService("secret")
	tok, err := ts.Sign(7, "admin", "Admin", true, time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := ts.Verify(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.UserID != 7 || claims.Username != "admin" || claims.Name != "Admin" || !claims.IsAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyExpired(t *testing.T) {
	ts := NewTokenService("secret")
	tok, err := ts.Sign(1, "u", "", false, -time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := ts.Verify(tok); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	tok, err := NewTokenService("a").Sign(1, "u", "", false, time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := NewTokenService("b").Verify(tok); err == nil {
		t.Fatal("expected error for mismatched secret")
	}
}
