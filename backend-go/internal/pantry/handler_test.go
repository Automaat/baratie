package pantry

import "testing"

func TestValidateRejects(t *testing.T) {
	if validate(&createRequest{Name: "  "}) == nil {
		t.Fatal("blank name should fail")
	}
	if validate(&createRequest{Name: "Flour", Quantity: -1}) == nil {
		t.Fatal("negative quantity should fail")
	}
	bad := "2026-13-40"
	if validate(&createRequest{Name: "Flour", ExpiresOn: &bad}) == nil {
		t.Fatal("bad expiry date should fail")
	}
}

func TestValidateDefaultsCategory(t *testing.T) {
	req := createRequest{Name: "  Flour  ", Quantity: 2, Unit: " kg ", Category: ""}
	if vErr := validate(&req); vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	it := toItem(&req)
	if it.Name != "Flour" || it.Unit != "kg" {
		t.Fatalf("not trimmed: %+v", it)
	}
	if it.Category != "other" {
		t.Fatalf("category = %q, want other", it.Category)
	}
}

func TestValidateParsesExpiry(t *testing.T) {
	d := "2026-03-15"
	req := createRequest{Name: "Milk", ExpiresOn: &d}
	if vErr := validate(&req); vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	it := toItem(&req)
	if it.ExpiresOn == nil || it.ExpiresOn.Format("2006-01-02") != d {
		t.Fatalf("expiry not parsed: %v", it.ExpiresOn)
	}
}
