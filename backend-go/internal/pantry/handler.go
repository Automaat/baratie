package pantry

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
	"github.com/Automaat/baratie/backend-go/internal/wire"
)

// response is the JSON shape returned for a pantry item.
type response struct {
	ID        int           `json:"id"`
	Name      string        `json:"name"`
	Quantity  float64       `json:"quantity"`
	Unit      string        `json:"unit"`
	Category  string        `json:"category"`
	ExpiresOn *wire.IsoDate `json:"expires_on"`
	CreatedAt wire.IsoNaive `json:"created_at"`
}

// createRequest is the body accepted by POST and PUT. expires holds the parsed
// expires_on value; validate populates it so toItem never has to fail.
type createRequest struct {
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	Category  string  `json:"category"`
	ExpiresOn *string `json:"expires_on"`

	expires *time.Time
}

// Handler is the HTTP boundary for /api/pantry.
type Handler struct {
	store  *Store
	logger *slog.Logger
}

// NewHandler wires the store and logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

func toResponse(it *Item) response {
	out := response{
		ID:        it.ID,
		Name:      it.Name,
		Quantity:  it.Quantity,
		Unit:      it.Unit,
		Category:  it.Category,
		CreatedAt: wire.IsoNaive(it.CreatedAt),
	}
	if it.ExpiresOn != nil {
		d := wire.IsoDate(*it.ExpiresOn)
		out.ExpiresOn = &d
	}
	return out
}

// validate checks and normalizes the request in place (trimming strings,
// defaulting the category, parsing the optional expiry), returning the first
// failure as a 422-shaped ValidationError.
func validate(req *createRequest) *httputil.ValidationError {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
	}
	if req.Quantity < 0 {
		return &httputil.ValidationError{Field: "quantity", Msg: "Quantity cannot be negative"}
	}
	req.Unit = strings.TrimSpace(req.Unit)
	req.Category = strings.TrimSpace(req.Category)
	if req.Category == "" {
		req.Category = "other"
	}
	if req.ExpiresOn != nil && strings.TrimSpace(*req.ExpiresOn) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*req.ExpiresOn))
		if err != nil {
			return &httputil.ValidationError{Field: "expires_on", Msg: "must be YYYY-MM-DD"}
		}
		req.expires = &parsed
	}
	return nil
}

// toItem builds an Item from an already-validated request.
func toItem(req *createRequest) *Item {
	return &Item{
		Name:      req.Name,
		Quantity:  req.Quantity,
		Unit:      req.Unit,
		Category:  req.Category,
		ExpiresOn: req.expires,
	}
}

// List serves GET /api/pantry.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list pantry", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]response, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Create serves POST /api/pantry.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httputil.DecodeJSON(w, r, 1<<16, &req) {
		return
	}
	if vErr := validate(&req); vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	created, err := h.store.Create(r.Context(), toItem(&req))
	if err != nil {
		h.logger.Error("create pantry item", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

// Update serves PUT /api/pantry/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "item_id")
	if !ok {
		return
	}
	var req createRequest
	if !httputil.DecodeJSON(w, r, 1<<16, &req) {
		return
	}
	if vErr := validate(&req); vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	updated, err := h.store.Update(r.Context(), id, toItem(&req))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Pantry item not found")
			return
		}
		h.logger.Error("update pantry item", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(updated))
}

// Delete serves DELETE /api/pantry/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "item_id")
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Pantry item not found")
			return
		}
		h.logger.Error("delete pantry item", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
