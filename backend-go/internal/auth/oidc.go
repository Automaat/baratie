package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
)

// OIDCStateCookie carries the signed state, nonce and PKCE verifier between
// the login redirect and the callback.
const OIDCStateCookie = "brt_oidc_state"

const (
	oidcCookiePath  = "/api/auth/oidc"
	oidcStateTTL    = 10 * time.Minute
	oidcHTTPTimeout = 10 * time.Second
)

// Error codes appended to /login?error=... for the login page to render.
const (
	OIDCErrNoUser      = "oidc_no_user"
	OIDCErrFailed      = "oidc_failed"
	OIDCErrUnavailable = "oidc_unavailable"
)

// UserLookup resolves an existing account by exact username; ErrNotFound when
// absent. *Store satisfies it.
type UserLookup interface {
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// OIDCHandler serves the OpenID Connect authorization-code flow (PKCE S256,
// client_secret_basic) under /api/auth/oidc. It never creates users: the
// provider's preferred_username must match an existing account.
type OIDCHandler struct {
	cfg          OIDCConfig
	users        UserLookup
	tokens       *TokenService
	stateKey     []byte
	cookieSecure bool
	client       *http.Client
	logger       *slog.Logger

	mu       sync.Mutex
	provider *oidc.Provider
}

// NewOIDCHandler wires the OIDC flow. Provider discovery is deferred to the
// first login so the backend starts (and password login works) while the
// provider is down. The state cookie key is derived from the JWT secret.
func NewOIDCHandler(cfg OIDCConfig, users UserLookup, tokens *TokenService, cookieSecure bool, logger *slog.Logger) *OIDCHandler {
	if logger == nil {
		logger = slog.Default()
	}
	mac := hmac.New(sha256.New, tokens.secret)
	mac.Write([]byte("baratie oidc state cookie v1"))
	return &OIDCHandler{
		cfg:          cfg,
		users:        users,
		tokens:       tokens,
		stateKey:     mac.Sum(nil),
		cookieSecure: cookieSecure,
		client:       &http.Client{Timeout: oidcHTTPTimeout},
		logger:       logger,
	}
}

type oidcStatusResponse struct {
	Enabled bool `json:"enabled"`
}

// Status serves GET /api/auth/oidc/status (public).
func (h *OIDCHandler) Status(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, oidcStatusResponse{Enabled: h.cfg.Enabled()})
}

// Login serves GET /api/auth/oidc/login (public): stores state, nonce and the
// PKCE verifier in a signed short-lived cookie and redirects to the provider.
func (h *OIDCHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Enabled() {
		httputil.WriteDetailError(w, http.StatusNotFound, "OIDC login is not configured")
		return
	}
	provider, err := h.discover(r.Context())
	if err != nil {
		h.logger.Warn("oidc login: provider unavailable", "err", err)
		redirectToLogin(w, r, OIDCErrUnavailable)
		return
	}
	st := oidcState{
		State:    rand.Text(),
		Nonce:    rand.Text(),
		Verifier: oauth2.GenerateVerifier(),
		Expires:  time.Now().Add(oidcStateTTL).Unix(),
	}
	value, err := h.encodeState(st)
	if err != nil {
		h.logger.Error("oidc login: encode state", "err", err)
		redirectToLogin(w, r, OIDCErrFailed)
		return
	}
	h.setStateCookie(w, value, int(oidcStateTTL.Seconds()))
	target := h.oauth2Config(provider).AuthCodeURL(st.State, oidc.Nonce(st.Nonce), oauth2.S256ChallengeOption(st.Verifier))
	http.Redirect(w, r, target, http.StatusFound)
}

// Callback serves GET /api/auth/oidc/callback (public). On success it issues
// the same session as password login and redirects to /; on any failure it
// redirects to /login?error=<code> and logs the cause.
func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Enabled() {
		httputil.WriteDetailError(w, http.StatusNotFound, "OIDC login is not configured")
		return
	}
	h.setStateCookie(w, "", -1)
	user, err := h.resolveUser(r)
	if err != nil {
		code := OIDCErrFailed
		var cbErr *callbackError
		if errors.As(err, &cbErr) {
			code = cbErr.code
		}
		h.logger.Warn("oidc callback failed", "code", code, "err", err)
		redirectToLogin(w, r, code)
		return
	}
	if _, err := startSession(w, h.tokens, user, h.cookieSecure, false); err != nil {
		h.logger.Error("oidc callback: sign token", "err", err)
		redirectToLogin(w, r, OIDCErrFailed)
		return
	}
	h.logger.Info("oidc login", "username", user.Username)
	http.Redirect(w, r, "/", http.StatusFound)
}

// callbackError tags a callback failure with the /login error code it maps to.
type callbackError struct {
	code string
	err  error
}

func (e *callbackError) Error() string { return e.err.Error() }
func (e *callbackError) Unwrap() error { return e.err }

func callbackFail(code string, err error) error {
	return &callbackError{code: code, err: err}
}

// resolveUser validates the callback request end to end and returns the
// matching existing user.
func (h *OIDCHandler) resolveUser(r *http.Request) (*User, error) {
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		return nil, callbackFail(OIDCErrFailed, fmt.Errorf("provider returned error %q", e))
	}
	st, err := h.readState(r)
	if err != nil {
		return nil, callbackFail(OIDCErrFailed, err)
	}
	if subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(st.State)) != 1 {
		return nil, callbackFail(OIDCErrFailed, errors.New("state mismatch"))
	}
	code := q.Get("code")
	if code == "" {
		return nil, callbackFail(OIDCErrFailed, errors.New("missing authorization code"))
	}
	provider, err := h.discover(r.Context())
	if err != nil {
		return nil, callbackFail(OIDCErrUnavailable, err)
	}
	ctx := oidc.ClientContext(r.Context(), h.client)
	tok, err := h.oauth2Config(provider).Exchange(ctx, code, oauth2.VerifierOption(st.Verifier))
	if err != nil {
		var re *oauth2.RetrieveError
		if errors.As(err, &re) {
			return nil, callbackFail(OIDCErrFailed, fmt.Errorf("exchange code: %w", err))
		}
		return nil, callbackFail(OIDCErrUnavailable, fmt.Errorf("exchange code: %w", err))
	}
	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, callbackFail(OIDCErrFailed, errors.New("token response has no id_token"))
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: h.cfg.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return nil, callbackFail(OIDCErrFailed, fmt.Errorf("verify id token: %w", err))
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(st.Nonce)) != 1 {
		return nil, callbackFail(OIDCErrFailed, errors.New("id token nonce mismatch"))
	}
	username, err := preferredUsername(ctx, provider, tok, idToken)
	if err != nil {
		return nil, callbackFail(OIDCErrFailed, err)
	}
	user, err := h.users.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, callbackFail(OIDCErrNoUser, fmt.Errorf("no user %q", username))
		}
		return nil, callbackFail(OIDCErrFailed, fmt.Errorf("look up user: %w", err))
	}
	return user, nil
}

type usernameClaims struct {
	PreferredUsername string `json:"preferred_username"`
}

// preferredUsername reads preferred_username from the ID token, falling back
// to the userinfo endpoint: Authelia omits profile claims from ID tokens
// unless a claims policy adds them.
func preferredUsername(ctx context.Context, provider *oidc.Provider, tok *oauth2.Token, idToken *oidc.IDToken) (string, error) {
	var claims usernameClaims
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("decode id token claims: %w", err)
	}
	if claims.PreferredUsername != "" {
		return claims.PreferredUsername, nil
	}
	info, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(tok))
	if err != nil {
		return "", fmt.Errorf("fetch userinfo: %w", err)
	}
	if info.Subject != idToken.Subject {
		return "", errors.New("userinfo subject does not match id token")
	}
	if err := info.Claims(&claims); err != nil {
		return "", fmt.Errorf("decode userinfo claims: %w", err)
	}
	if claims.PreferredUsername == "" {
		return "", errors.New("no preferred_username claim in id token or userinfo")
	}
	return claims.PreferredUsername, nil
}

// discover returns the cached provider, running discovery on first use. A
// failure is not cached, so the next login retries.
func (h *OIDCHandler) discover(ctx context.Context) (*oidc.Provider, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.provider != nil {
		return h.provider, nil
	}
	ctx, cancel := context.WithTimeout(oidc.ClientContext(ctx, h.client), oidcHTTPTimeout)
	defer cancel()
	provider, err := oidc.NewProvider(ctx, h.cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover %s: %w", h.cfg.IssuerURL, err)
	}
	h.provider = provider
	return provider, nil
}

func (h *OIDCHandler) oauth2Config(provider *oidc.Provider) *oauth2.Config {
	endpoint := provider.Endpoint()
	endpoint.AuthStyle = oauth2.AuthStyleInHeader
	return &oauth2.Config{
		ClientID:     h.cfg.ClientID,
		ClientSecret: h.cfg.ClientSecret,
		RedirectURL:  h.cfg.RedirectURL,
		Endpoint:     endpoint,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
}

// oidcState is the payload of the state cookie.
type oidcState struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Expires  int64  `json:"e"`
}

// encodeState serializes st as base64url(json) + "." + base64url(HMAC).
func (h *OIDCHandler) encodeState(st oidcState) (string, error) {
	raw, err := json.Marshal(st)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + base64.RawURLEncoding.EncodeToString(h.sign(payload)), nil
}

func (h *OIDCHandler) readState(r *http.Request) (oidcState, error) {
	c, err := r.Cookie(OIDCStateCookie)
	if err != nil {
		return oidcState{}, errors.New("missing state cookie")
	}
	payload, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return oidcState{}, errors.New("malformed state cookie")
	}
	gotSig, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(gotSig, h.sign(payload)) {
		return oidcState{}, errors.New("state cookie signature mismatch")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return oidcState{}, fmt.Errorf("decode state cookie: %w", err)
	}
	var st oidcState
	if err := json.Unmarshal(raw, &st); err != nil {
		return oidcState{}, fmt.Errorf("decode state cookie: %w", err)
	}
	if time.Now().Unix() > st.Expires {
		return oidcState{}, errors.New("state cookie expired")
	}
	return st, nil
}

func (h *OIDCHandler) sign(payload string) []byte {
	mac := hmac.New(sha256.New, h.stateKey)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func (h *OIDCHandler) setStateCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     OIDCStateCookie,
		Value:    value,
		Path:     oidcCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func redirectToLogin(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?error="+code, http.StatusFound)
}
