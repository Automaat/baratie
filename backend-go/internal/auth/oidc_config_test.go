package auth

import (
	"maps"
	"strings"
	"testing"
)

const (
	envIssuer    = "BRT_OIDC_ISSUER_URL"
	envClientID  = "BRT_OIDC_CLIENT_ID"
	envClientKey = "BRT_OIDC_CLIENT_SECRET"
	envRedirect  = "BRT_OIDC_REDIRECT_URL"
)

func TestParseOIDCConfig(t *testing.T) {
	full := map[string]string{
		envIssuer:    "https://auth.example.test",
		envClientID:  "baratie",
		envClientKey: "s3cret",
		envRedirect:  "https://baratie.example.test/api/auth/oidc/callback",
	}
	without := func(keys ...string) map[string]string {
		out := maps.Clone(full)
		for _, k := range keys {
			delete(out, k)
		}
		return out
	}
	with := func(key, value string) map[string]string {
		out := without()
		out[key] = value
		return out
	}

	tests := []struct {
		name        string
		env         map[string]string
		wantEnabled bool
		wantErr     []string
	}{
		{name: "all set", env: full, wantEnabled: true},
		{name: "none set", env: map[string]string{}},
		{
			name: "all blank counts as none",
			env: map[string]string{
				envIssuer: " ", envClientID: "", envClientKey: "", envRedirect: "",
			},
		},
		{name: "missing secret", env: without(envClientKey), wantErr: []string{envClientKey}},
		{
			name:    "only issuer",
			env:     map[string]string{envIssuer: "https://auth.example.test"},
			wantErr: []string{envClientID, envClientKey, envRedirect},
		},
		{name: "relative issuer", env: with(envIssuer, "auth.example.test"), wantErr: []string{envIssuer}},
		{name: "non-http redirect", env: with(envRedirect, "ftp://x/cb"), wantErr: []string{envRedirect}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseOIDCConfig(func(k string) string { return tt.env[k] })
			if len(tt.wantErr) > 0 {
				if err == nil {
					t.Fatalf("ParseOIDCConfig() error = nil, want error naming %v", tt.wantErr)
				}
				for _, want := range tt.wantErr {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q does not name %s", err, want)
					}
				}
				if cfg.Enabled() {
					t.Error("config enabled despite error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseOIDCConfig() error = %v", err)
			}
			if cfg.Enabled() != tt.wantEnabled {
				t.Fatalf("Enabled() = %v, want %v", cfg.Enabled(), tt.wantEnabled)
			}
			if tt.wantEnabled && cfg.ClientSecret != full[envClientKey] {
				t.Errorf("ClientSecret = %q, want %q", cfg.ClientSecret, full[envClientKey])
			}
		})
	}
}
