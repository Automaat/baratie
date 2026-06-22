package auth

import "testing"

func TestValidateUsername(t *testing.T) {
	if _, msg := validateUsername("   "); msg == "" {
		t.Fatal("blank username should fail")
	}
	u, msg := validateUsername("  bob  ")
	if msg != "" || u != "bob" {
		t.Fatalf("got (%q, %q), want (bob, )", u, msg)
	}
}

func TestValidatePassword(t *testing.T) {
	if validatePassword("short") == "" {
		t.Fatal("short password should fail")
	}
	if validatePassword("longenough") != "" {
		t.Fatal("8+ char password should pass")
	}
}

func TestValidateNames(t *testing.T) {
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	s := string(long)
	if _, _, msg := validateNames(&s, nil); msg == "" {
		t.Fatal("over-long name should fail")
	}
	rawName, rawSurname := "  Ann  ", "  "
	name, surname, msg := validateNames(&rawName, &rawSurname)
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if name == nil || *name != "Ann" {
		t.Fatalf("name not trimmed: %v", name)
	}
	if surname != nil {
		t.Fatalf("blank surname should be nil, got %v", surname)
	}
}
