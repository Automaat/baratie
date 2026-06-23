package httputil

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// DecodeJSON reads a size-limited JSON request body and writes the validation
// envelope on decode failure.
func DecodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst any) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBytes)).Decode(dst); err != nil {
		WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return false
	}
	return true
}

// PathInt parses an integer chi path parameter and writes the validation
// envelope on parse failure.
func PathInt(w http.ResponseWriter, r *http.Request, param string) (int, bool) {
	return PathIntField(w, r, param, param)
}

// PathIntField parses an integer chi path parameter and writes a validation
// envelope under field. Use when the URL placeholder differs from the error
// field that clients/tests expect.
func PathIntField(w http.ResponseWriter, r *http.Request, param, field string) (int, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.Atoi(raw)
	if err != nil {
		WriteBodyValidationError(w, field, "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// OptionalDate parses an optional YYYY-MM-DD query value. An empty value yields
// (nil, true) — an open bound. A malformed value writes a 422 and returns
// (nil, false) so the caller can stop.
func OptionalDate(w http.ResponseWriter, raw, field string) (*time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, true
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		WriteBodyValidationError(w, field, "must be YYYY-MM-DD", raw)
		return nil, false
	}
	return &t, true
}
