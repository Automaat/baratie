package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testClientID     = "baratie"
	testClientSecret = "test-client-s3cret"
	testAccessToken  = "fake-access-token"
	testSubject      = "user-sub-123"
	testKeyID        = "test-key"
	testRedirectURL  = "https://baratie.example.test/api/auth/oidc/callback"
)

var testSigningKey = sync.OnceValue(func() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
})

type issuedCode struct {
	challenge string
	nonce     string
}

// fakeProvider is an in-process OIDC provider: discovery, JWKS, token
// endpoint (client_secret_basic + PKCE S256) and userinfo. discoveryDown makes
// discovery answer 503 to simulate an outage.
type fakeProvider struct {
	t             *testing.T
	srv           *httptest.Server
	discoveryDown atomic.Bool

	mu               sync.Mutex
	codes            map[string]issuedCode
	username         string
	omitIDTokenName  bool
	nonceOverride    string
	userinfoUsername string
}

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	f := &fakeProvider{t: t, codes: map[string]issuedCode{}, username: "mskalski"}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", f.discovery)
	mux.HandleFunc("GET /jwks", f.jwks)
	mux.HandleFunc("POST /token", f.token)
	mux.HandleFunc("GET /userinfo", f.userinfo)
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeProvider) discovery(w http.ResponseWriter, _ *http.Request) {
	if f.discoveryDown.Load() {
		http.Error(w, "down", http.StatusServiceUnavailable)
		return
	}
	writeTestJSON(w, map[string]any{
		"issuer":                                f.srv.URL,
		"authorization_endpoint":                f.srv.URL + "/authorize",
		"token_endpoint":                        f.srv.URL + "/token",
		"jwks_uri":                              f.srv.URL + "/jwks",
		"userinfo_endpoint":                     f.srv.URL + "/userinfo",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (f *fakeProvider) jwks(w http.ResponseWriter, _ *http.Request) {
	pub := testSigningKey().PublicKey
	writeTestJSON(w, map[string]any{"keys": []map[string]string{{
		"kty": "RSA",
		"kid": testKeyID,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}})
}

func (f *fakeProvider) token(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, secret, ok := r.BasicAuth()
	if !ok || id != testClientID || secret != testClientSecret {
		w.WriteHeader(http.StatusUnauthorized)
		writeTestJSON(w, map[string]string{"error": "invalid_client"})
		return
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	issued, found := f.codes[r.PostForm.Get("code")]
	delete(f.codes, r.PostForm.Get("code"))
	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	if !found || base64.RawURLEncoding.EncodeToString(sum[:]) != issued.challenge ||
		r.PostForm.Get("redirect_uri") != testRedirectURL {
		w.WriteHeader(http.StatusBadRequest)
		writeTestJSON(w, map[string]string{"error": "invalid_grant"})
		return
	}
	nonce := issued.nonce
	if f.nonceOverride != "" {
		nonce = f.nonceOverride
	}
	claims := jwt.MapClaims{
		"iss":   f.srv.URL,
		"aud":   testClientID,
		"sub":   testSubject,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(5 * time.Minute).Unix(),
		"nonce": nonce,
	}
	if !f.omitIDTokenName {
		claims["preferred_username"] = f.username
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKeyID
	signed, err := tok.SignedString(testSigningKey())
	if err != nil {
		f.t.Errorf("sign id token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeTestJSON(w, map[string]any{
		"access_token": testAccessToken,
		"token_type":   "Bearer",
		"expires_in":   300,
		"id_token":     signed,
	})
}

func (f *fakeProvider) userinfo(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+testAccessToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	writeTestJSON(w, map[string]string{"sub": testSubject, "preferred_username": f.userinfoUsername})
}

// authorize plays the provider's authorization endpoint: it checks the
// redirect built by Login and issues a code bound to its PKCE challenge and
// nonce.
func (f *fakeProvider) authorize(t *testing.T, location string) (string, string) {
	t.Helper()
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse login redirect: %v", err)
	}
	if got := u.Scheme + "://" + u.Host + u.Path; got != f.srv.URL+"/authorize" {
		t.Fatalf("login redirect = %s, want provider authorize endpoint", got)
	}
	q := u.Query()
	want := map[string]string{
		"client_id":             testClientID,
		"response_type":         "code",
		"scope":                 "openid profile email",
		"redirect_uri":          testRedirectURL,
		"code_challenge_method": "S256",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("authorize param %s = %q, want %q", k, q.Get(k), v)
		}
	}
	for _, k := range []string{"state", "nonce", "code_challenge"} {
		if q.Get(k) == "" {
			t.Errorf("authorize param %s is empty", k)
		}
	}
	code := rand.Text()
	f.mu.Lock()
	f.codes[code] = issuedCode{challenge: q.Get("code_challenge"), nonce: q.Get("nonce")}
	f.mu.Unlock()
	return code, q.Get("state")
}

func writeTestJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

type fakeUsers map[string]*User

func (f fakeUsers) GetByUsername(_ context.Context, username string) (*User, error) {
	if u, ok := f[username]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}

type oidcEnv struct {
	provider *fakeProvider
	app      http.Handler
	logs     *bytes.Buffer
}

func newOIDCEnv(t *testing.T, cookieSecure bool) *oidcEnv {
	t.Helper()
	provider := newFakeProvider(t)
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tokens := NewTokenService("test-jwt-secret")
	users := fakeUsers{"mskalski": {ID: 7, Username: "mskalski", IsAdmin: true}}
	cfg := OIDCConfig{
		IssuerURL:    provider.srv.URL,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirectURL,
	}
	h := NewOIDCHandler(cfg, users, tokens, cookieSecure, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/oidc/status", h.Status)
	mux.HandleFunc("GET /api/auth/oidc/login", h.Login)
	mux.HandleFunc("GET /api/auth/oidc/callback", h.Callback)
	mux.Handle("GET /api/protected", Authenticate(tokens, nil, logger)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			claims, _ := claimsFrom(r.Context())
			_, _ = w.Write([]byte(claims.Username))
		})))
	return &oidcEnv{provider: provider, app: mux, logs: logs}
}

func (e *oidcEnv) get(t *testing.T, target string, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, http.NoBody)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	e.app.ServeHTTP(rec, req)
	return rec.Result()
}

func findCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// startLogin runs the login endpoint and returns the provider redirect plus
// the state cookie the browser would store.
func (e *oidcEnv) startLogin(t *testing.T) (string, *http.Cookie) {
	t.Helper()
	resp := e.get(t, "/api/auth/oidc/login")
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("login status = %d, want 302", resp.StatusCode)
	}
	c := findCookie(resp, OIDCStateCookie)
	if c == nil {
		t.Fatal("login set no state cookie")
	}
	return resp.Header.Get("Location"), c
}

func TestOIDCLoginRedirect(t *testing.T) {
	env := newOIDCEnv(t, true)
	location, c := env.startLogin(t)
	env.provider.authorize(t, location)

	if c.Path != "/api/auth/oidc" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
		t.Errorf("state cookie attrs = path %q httpOnly %v secure %v sameSite %v",
			c.Path, c.HttpOnly, c.Secure, c.SameSite)
	}
	if c.MaxAge != 600 {
		t.Errorf("state cookie MaxAge = %d, want 600", c.MaxAge)
	}
}

func keepCookie(_ *oidcEnv, _ url.Values, c *http.Cookie) []*http.Cookie {
	return []*http.Cookie{c}
}

func TestOIDCCallback(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(f *fakeProvider)
		tamper       func(env *oidcEnv, q url.Values, c *http.Cookie) []*http.Cookie
		wantLocation string
		wantUser     string
	}{
		{name: "existing user", wantLocation: "/", wantUser: "mskalski"},
		{
			name: "username from userinfo",
			setup: func(f *fakeProvider) {
				f.omitIDTokenName = true
				f.userinfoUsername = "mskalski"
			},
			wantLocation: "/",
			wantUser:     "mskalski",
		},
		{
			name:         "unknown user",
			setup:        func(f *fakeProvider) { f.username = "stranger" },
			wantLocation: "/login?error=oidc_no_user",
		},
		{
			name: "bad state",
			tamper: func(env *oidcEnv, q url.Values, c *http.Cookie) []*http.Cookie {
				q.Set("state", "forged")
				return keepCookie(env, q, c)
			},
			wantLocation: "/login?error=oidc_failed",
		},
		{
			name:         "nonce mismatch",
			setup:        func(f *fakeProvider) { f.nonceOverride = "other-nonce" },
			wantLocation: "/login?error=oidc_failed",
		},
		{
			name:         "missing cookie",
			tamper:       func(*oidcEnv, url.Values, *http.Cookie) []*http.Cookie { return nil },
			wantLocation: "/login?error=oidc_failed",
		},
		{
			name: "tampered cookie",
			tamper: func(_ *oidcEnv, _ url.Values, c *http.Cookie) []*http.Cookie {
				payload, _, _ := strings.Cut(c.Value, ".")
				return []*http.Cookie{{Name: c.Name, Value: payload + ".AAAA"}}
			},
			wantLocation: "/login?error=oidc_failed",
		},
		{
			name: "provider error response",
			tamper: func(env *oidcEnv, q url.Values, c *http.Cookie) []*http.Cookie {
				q.Del("code")
				q.Set("error", "access_denied")
				return keepCookie(env, q, c)
			},
			wantLocation: "/login?error=oidc_failed",
		},
		{
			name: "provider down",
			tamper: func(env *oidcEnv, q url.Values, c *http.Cookie) []*http.Cookie {
				env.provider.srv.Close()
				return keepCookie(env, q, c)
			},
			wantLocation: "/login?error=oidc_unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newOIDCEnv(t, false)
			if tt.setup != nil {
				tt.setup(env.provider)
			}
			location, stateCookie := env.startLogin(t)
			code, state := env.provider.authorize(t, location)
			q := url.Values{"code": {code}, "state": {state}}
			tamper := tt.tamper
			if tamper == nil {
				tamper = keepCookie
			}
			cookies := tamper(env, q, stateCookie)

			resp := env.get(t, "/api/auth/oidc/callback?"+q.Encode(), cookies...)

			if resp.StatusCode != http.StatusFound {
				t.Fatalf("callback status = %d, want 302", resp.StatusCode)
			}
			if got := resp.Header.Get("Location"); got != tt.wantLocation {
				t.Fatalf("callback Location = %q, want %q", got, tt.wantLocation)
			}
			if c := findCookie(resp, OIDCStateCookie); c == nil || c.MaxAge >= 0 {
				t.Errorf("state cookie not cleared: %+v", c)
			}
			for _, secret := range []string{testClientSecret, testAccessToken, code} {
				if strings.Contains(env.logs.String(), secret) {
					t.Errorf("logs leak %q:\n%s", secret, env.logs.String())
				}
			}
			assertSession(t, env, resp, tt.wantUser)
		})
	}
}

// assertSession checks the callback's session cookie: present and accepted by
// a protected endpoint when wantUser is set, absent otherwise.
func assertSession(t *testing.T, env *oidcEnv, resp *http.Response, wantUser string) {
	t.Helper()
	session := findCookie(resp, CookieName)
	if wantUser == "" {
		if session != nil {
			t.Fatalf("failed callback set session cookie %q", session.Value)
		}
		return
	}
	if session == nil || session.Value == "" {
		t.Fatal("successful callback set no session cookie")
	}
	if session.Path != "/" || !session.HttpOnly || session.SameSite != http.SameSiteLaxMode || session.MaxAge != 0 {
		t.Errorf("session cookie attrs differ from password login: %+v", session)
	}
	protected := env.get(t, "/api/protected", session)
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(protected.Body)
	if protected.StatusCode != http.StatusOK || body.String() != wantUser {
		t.Fatalf("protected endpoint = %d %q, want 200 %q", protected.StatusCode, body.String(), wantUser)
	}
}

func TestOIDCLoginProviderDown(t *testing.T) {
	env := newOIDCEnv(t, false)
	env.provider.srv.Close()

	resp := env.get(t, "/api/auth/oidc/login")

	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/login?error=oidc_unavailable" {
		t.Fatalf("login = %d %q, want 302 to oidc_unavailable", resp.StatusCode, resp.Header.Get("Location"))
	}
	if findCookie(resp, OIDCStateCookie) != nil {
		t.Error("login set a state cookie although the provider is down")
	}
}

// TestOIDCDiscoveryRetried pins lazy discovery: a failed discovery is not
// cached, so login works once the provider is back.
func TestOIDCDiscoveryRetried(t *testing.T) {
	env := newOIDCEnv(t, false)
	env.provider.discoveryDown.Store(true)
	if loc := env.get(t, "/api/auth/oidc/login").Header.Get("Location"); loc != "/login?error=oidc_unavailable" {
		t.Fatalf("login while down redirected to %q", loc)
	}

	env.provider.discoveryDown.Store(false)
	location, _ := env.startLogin(t)
	env.provider.authorize(t, location)
}

func TestOIDCStatus(t *testing.T) {
	tests := []struct {
		name string
		cfg  OIDCConfig
		want string
	}{
		{name: "enabled", cfg: OIDCConfig{IssuerURL: "https://auth.example.test", ClientID: "x"}, want: `{"enabled":true}`},
		{name: "disabled", cfg: OIDCConfig{}, want: `{"enabled":false}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewOIDCHandler(tt.cfg, fakeUsers{}, NewTokenService("s"), false, nil)
			rec := httptest.NewRecorder()
			h.Status(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody))
			if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != tt.want {
				t.Fatalf("status = %d %s, want 200 %s", rec.Code, got, tt.want)
			}
		})
	}
}

func TestOIDCDisabledEndpoints(t *testing.T) {
	h := NewOIDCHandler(OIDCConfig{}, fakeUsers{}, NewTokenService("s"), false, nil)
	for name, fn := range map[string]http.HandlerFunc{"login": h.Login, "callback": h.Callback} {
		rec := httptest.NewRecorder()
		fn(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", name, rec.Code)
		}
	}
}
