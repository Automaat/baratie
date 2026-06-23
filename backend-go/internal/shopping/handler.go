package shopping

import (
	"log/slog"
	"net/http"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
)

// itemResponse is the JSON shape for one consolidated shopping-list line.
// amount/unit are populated for structured ingredients (summed per food/unit)
// and zero/empty for free-form ingredients.
type itemResponse struct {
	Name     string   `json:"name"`
	Amount   float64  `json:"amount"`
	Unit     string   `json:"unit"`
	Recipes  []string `json:"recipes"`
	InPantry bool     `json:"in_pantry"`
}

// listResponse is the body returned by GET /api/shopping-list.
type listResponse struct {
	Items []itemResponse `json:"items"`
	Count int            `json:"count"`
}

// Handler is the HTTP boundary for /api/shopping-list.
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

// List serves GET /api/shopping-list with optional date_from / date_to filters.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, ok := httputil.OptionalDate(w, q.Get("date_from"), "date_from")
	if !ok {
		return
	}
	to, ok := httputil.OptionalDate(w, q.Get("date_to"), "date_to")
	if !ok {
		return
	}

	structured, err := h.store.PlannedStructured(r.Context(), from, to)
	if err != nil {
		h.logger.Error("shopping list: structured ingredients", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	planned, err := h.store.PlannedRecipes(r.Context(), from, to)
	if err != nil {
		h.logger.Error("shopping list: planned recipes", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	pantry, err := h.store.PantryNames(r.Context())
	if err != nil {
		h.logger.Error("shopping list: pantry names", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(build(structured, planned, pantry)))
}

func toResponse(items []Item) listResponse {
	out := make([]itemResponse, 0, len(items))
	for _, it := range items {
		// Item and itemResponse share an identical field sequence; Go ignores
		// the JSON tags in the conversion (staticcheck S1016).
		out = append(out, itemResponse(it))
	}
	return listResponse{Items: out, Count: len(out)}
}
