package foods

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
	"github.com/Automaat/baratie/backend-go/internal/wire"
)

// response is the JSON shape returned for a food.
type response struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	KcalPer100g    float64       `json:"kcal_per_100g"`
	ProteinPer100g float64       `json:"protein_per_100g"`
	CarbsPer100g   float64       `json:"carbs_per_100g"`
	FatPer100g     float64       `json:"fat_per_100g"`
	CreatedAt      wire.IsoNaive `json:"created_at"`
}

// createRequest is the body accepted by POST and PUT.
type createRequest struct {
	Name           string  `json:"name"`
	KcalPer100g    float64 `json:"kcal_per_100g"`
	ProteinPer100g float64 `json:"protein_per_100g"`
	CarbsPer100g   float64 `json:"carbs_per_100g"`
	FatPer100g     float64 `json:"fat_per_100g"`
}

// Handler is the HTTP boundary for /api/foods.
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

func toResponse(f *Food) response {
	return response{
		ID:             f.ID,
		Name:           f.Name,
		KcalPer100g:    f.KcalPer100g,
		ProteinPer100g: f.ProteinPer100g,
		CarbsPer100g:   f.CarbsPer100g,
		FatPer100g:     f.FatPer100g,
		CreatedAt:      wire.IsoNaive(f.CreatedAt),
	}
}

func toFood(req *createRequest) *Food {
	return &Food{
		Name:           req.Name,
		KcalPer100g:    req.KcalPer100g,
		ProteinPer100g: req.ProteinPer100g,
		CarbsPer100g:   req.CarbsPer100g,
		FatPer100g:     req.FatPer100g,
	}
}

// List serves GET /api/foods.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list foods", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]response, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Create serves POST /api/foods.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httputil.DecodeJSON(w, r, 1<<16, &req) {
		return
	}
	if vErr := validate(&req); vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	created, err := h.store.Create(r.Context(), toFood(&req))
	if err != nil {
		h.writeStoreError(w, err, "create food")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

// Update serves PUT /api/foods/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "food_id")
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
	updated, err := h.store.Update(r.Context(), id, toFood(&req))
	if err != nil {
		h.writeStoreError(w, err, "update food")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(updated))
}

// Delete serves DELETE /api/foods/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "food_id")
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, err, "delete food")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeStoreError maps store sentinels to HTTP status codes.
func (h *Handler) writeStoreError(w http.ResponseWriter, err error, op string) {
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound, "Food not found")
	case errors.Is(err, ErrNameConflict):
		httputil.WriteDetailError(w, http.StatusConflict, "A food with that name already exists")
	case errors.Is(err, ErrInUse):
		httputil.WriteDetailError(w, http.StatusConflict, "Food is used by a recipe and cannot be deleted")
	default:
		h.logger.Error(op, "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

// validate checks and normalizes the request in place.
func validate(req *createRequest) *httputil.ValidationError {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
	}
	if len(req.Name) > 200 {
		return &httputil.ValidationError{Field: "name", Msg: "Name too long (max 200 characters)"}
	}
	macros := []struct {
		field string
		value float64
		label string
	}{
		{"kcal_per_100g", req.KcalPer100g, "Calories"},
		{"protein_per_100g", req.ProteinPer100g, "Protein"},
		{"carbs_per_100g", req.CarbsPer100g, "Carbs"},
		{"fat_per_100g", req.FatPer100g, "Fat"},
	}
	for _, m := range macros {
		if m.value < 0 {
			return &httputil.ValidationError{Field: m.field, Msg: m.label + " cannot be negative"}
		}
	}
	return nil
}
