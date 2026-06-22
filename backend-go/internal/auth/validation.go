package auth

import "strings"

const minPasswordLen = 8

// validateUsername trims and bounds the username. The second return value is
// a non-empty message when validation fails.
func validateUsername(raw string) (string, string) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", "Username cannot be empty"
	}
	if len(u) > 100 {
		return "", "Username too long (max 100 characters)"
	}
	return u, ""
}

// validatePassword returns a non-empty message when the password is too short.
func validatePassword(raw string) string {
	if len(raw) < minPasswordLen {
		return "Password must be at least 8 characters"
	}
	return ""
}

// optionalName trims an optional name/surname; an empty string becomes nil.
// field names the value for the error message.
func optionalName(raw *string, field string) (*string, string) {
	if raw == nil {
		return nil, ""
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, ""
	}
	if len(trimmed) > 100 {
		return nil, field + " too long (max 100 characters)"
	}
	return &trimmed, ""
}

// validateNames trims name + surname; returns the first non-empty error.
func validateNames(rawName, rawSurname *string) (*string, *string, string) {
	name, vErr := optionalName(rawName, "Name")
	if vErr != "" {
		return nil, nil, vErr
	}
	surname, vErr := optionalName(rawSurname, "Surname")
	if vErr != "" {
		return nil, nil, vErr
	}
	return name, surname, ""
}
