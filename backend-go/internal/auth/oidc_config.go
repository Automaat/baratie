package auth

import (
	"fmt"
	"net/url"
	"strings"
)

// OIDCConfig is the OpenID Connect client configuration. The zero value means
// OIDC login is disabled.
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Enabled reports whether OIDC login is configured. ParseOIDCConfig only ever
// returns all-set or all-empty configs, so one field is enough.
func (c OIDCConfig) Enabled() bool {
	return c.IssuerURL != ""
}

// ParseOIDCConfig reads BRT_OIDC_ISSUER_URL, BRT_OIDC_CLIENT_ID,
// BRT_OIDC_CLIENT_SECRET and BRT_OIDC_REDIRECT_URL through getenv (os.Getenv in
// production). It returns the zero config when none is set and an error naming
// the missing variables when only some are.
func ParseOIDCConfig(getenv func(string) string) (OIDCConfig, error) {
	var cfg OIDCConfig
	fields := []struct {
		env string
		dst *string
	}{
		{"BRT_OIDC_ISSUER_URL", &cfg.IssuerURL},
		{"BRT_OIDC_CLIENT_ID", &cfg.ClientID},
		{"BRT_OIDC_CLIENT_SECRET", &cfg.ClientSecret},
		{"BRT_OIDC_REDIRECT_URL", &cfg.RedirectURL},
	}
	all := make([]string, 0, len(fields))
	var missing []string
	for _, f := range fields {
		all = append(all, f.env)
		*f.dst = strings.TrimSpace(getenv(f.env))
		if *f.dst == "" {
			missing = append(missing, f.env)
		}
	}
	switch len(missing) {
	case 0:
	case len(fields):
		return OIDCConfig{}, nil
	default:
		return OIDCConfig{}, fmt.Errorf("OIDC is partially configured: missing %s (set all of %s, or none)",
			strings.Join(missing, ", "), strings.Join(all, ", "))
	}
	if err := validateAbsoluteURL("BRT_OIDC_ISSUER_URL", cfg.IssuerURL); err != nil {
		return OIDCConfig{}, err
	}
	if err := validateAbsoluteURL("BRT_OIDC_REDIRECT_URL", cfg.RedirectURL); err != nil {
		return OIDCConfig{}, err
	}
	return cfg, nil
}

func validateAbsoluteURL(name, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s is not a valid URL: %w", name, err)
	}
	if (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("%s must be an absolute http(s) URL, got %q", name, raw)
	}
	return nil
}
