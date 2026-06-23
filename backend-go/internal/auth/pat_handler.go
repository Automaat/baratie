package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
	"github.com/Automaat/baratie/backend-go/internal/wire"
)

// createTokenRequest is the body accepted by POST /api/auth/tokens. expiresAt
// holds the parsed expires_at value; validate populates it.
type createTokenRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at"`

	expiresAt *time.Time
}

// tokenResponse is the metadata shape returned for a token; it never carries
// the secret.
type tokenResponse struct {
	ID         int            `json:"id"`
	Name       string         `json:"name"`
	Scope      string         `json:"scope"`
	CreatedAt  wire.IsoNaive  `json:"created_at"`
	ExpiresAt  *wire.IsoNaive `json:"expires_at"`
	LastUsedAt *wire.IsoNaive `json:"last_used_at"`
}

// createTokenResponse adds the one-time plaintext secret to the metadata. It is
// returned only by Create — the secret is unrecoverable afterwards.
type createTokenResponse struct {
	Token string `json:"token"`
	tokenResponse
}

func toTokenResponse(p *PersonalAccessToken) tokenResponse {
	out := tokenResponse{
		ID:        p.ID,
		Name:      p.Name,
		Scope:     p.Scope,
		CreatedAt: wire.IsoNaive(p.CreatedAt),
	}
	if p.ExpiresAt != nil {
		v := wire.IsoNaive(*p.ExpiresAt)
		out.ExpiresAt = &v
	}
	if p.LastUsedAt != nil {
		v := wire.IsoNaive(*p.LastUsedAt)
		out.LastUsedAt = &v
	}
	return out
}

// ListTokens serves GET /api/auth/tokens — the caller's own tokens.
func (h *Handler) ListTokens(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tokens, err := h.pats.List(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("list tokens", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]tokenResponse, 0, len(tokens))
	for i := range tokens {
		out = append(out, toTokenResponse(&tokens[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// CreateToken serves POST /api/auth/tokens. It mints a token for the caller and
// returns the secret once.
func (h *Handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	var req createTokenRequest
	if !httputil.DecodeJSON(w, r, 1<<16, &req) {
		return
	}
	if vErr := validateToken(&req); vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	raw, pat, err := h.pats.Create(r.Context(), claims.UserID, req.Name, req.expiresAt)
	if err != nil {
		h.logger.Error("create token", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, createTokenResponse{
		Token:         raw,
		tokenResponse: toTokenResponse(pat),
	})
}

// RevokeToken serves DELETE /api/auth/tokens/{id} — revokes a token the caller
// owns.
func (h *Handler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	id, ok := httputil.PathIntField(w, r, "id", "token_id")
	if !ok {
		return
	}
	if err := h.pats.Delete(r.Context(), claims.UserID, id); err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Token not found")
			return
		}
		h.logger.Error("revoke token", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validateToken checks and normalizes the request in place: a required name and
// an optional future expiry date (YYYY-MM-DD), interpreted as end of that day.
func validateToken(req *createTokenRequest) *httputil.ValidationError {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
	}
	if len(req.Name) > 100 {
		return &httputil.ValidationError{Field: "name", Msg: "Name too long (max 100 characters)"}
	}
	if req.ExpiresAt == nil || strings.TrimSpace(*req.ExpiresAt) == "" {
		return nil
	}
	day, err := time.Parse("2006-01-02", strings.TrimSpace(*req.ExpiresAt))
	if err != nil {
		return &httputil.ValidationError{Field: "expires_at", Msg: "must be YYYY-MM-DD"}
	}
	// Valid through the whole chosen day: the last microsecond the timestamp
	// column resolves, so `expires_at > now()` keeps it live until day's end
	// without rolling the displayed date to the next day.
	expiry := day.Add(24*time.Hour - time.Microsecond)
	if !expiry.After(time.Now().UTC()) {
		return &httputil.ValidationError{Field: "expires_at", Msg: "must be in the future"}
	}
	req.expiresAt = &expiry
	return nil
}
