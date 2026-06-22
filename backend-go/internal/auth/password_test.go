package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("hunter2!!")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !checkPassword(hash, "hunter2!!") {
		t.Fatal("correct password should match")
	}
	if checkPassword(hash, "wrong") {
		t.Fatal("wrong password should not match")
	}
}
