package mealplan

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
	"github.com/Automaat/baratie/backend-go/internal/wire"
)

// validMealTypes bounds the meal_type field. Anything else is a 422.
var validMealTypes = map[string]struct{}{
	"breakfast": {}, "lunch": {}, "dinner": {}, "snack": {},
}

// response is the JSON shape returned for a meal plan entry.
type response struct {
	ID         int           `json:"id"`
	PlanDate   wire.IsoDate  `json:"plan_date"`
	MealType   string        `json:"meal_type"`
	RecipeID   *int          `json:"recipe_id"`
	RecipeName *string       `json:"recipe_name"`
	Note       string        `json:"note"`
	CreatedAt  wire.IsoNaive `json:"created_at"`
}

// createRequest is the body accepted by POST and PUT. planDate holds the
// parsed plan_date; validate populates it so toEntry never has to fail.
type createRequest struct {
	PlanDate string `json:"plan_date"`
	MealType string `json:"meal_type"`
	RecipeID *int   `json:"recipe_id"`
	Note     string `json:"note"`

	planDate time.Time
}

// Handler is the HTTP boundary for /api/meal-plan.
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

func toResponse(e *Entry) response {
	return response{
		ID:         e.ID,
		PlanDate:   wire.IsoDate(e.PlanDate),
		MealType:   e.MealType,
		RecipeID:   e.RecipeID,
		RecipeName: e.RecipeName,
		Note:       e.Note,
		CreatedAt:  wire.IsoNaive(e.CreatedAt),
	}
}

// validate checks and normalizes the request in place (parsing the date,
// bounding the meal type, trimming the note), returning the first failure as a
// 422-shaped ValidationError.
func validate(req *createRequest) *httputil.ValidationError {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(req.PlanDate))
	if err != nil {
		return &httputil.ValidationError{Field: "plan_date", Msg: "must be YYYY-MM-DD"}
	}
	req.planDate = date
	req.MealType = strings.TrimSpace(req.MealType)
	if _, ok := validMealTypes[req.MealType]; !ok {
		return &httputil.ValidationError{
			Field: "meal_type",
			Msg:   "must be one of breakfast, lunch, dinner, snack",
		}
	}
	req.Note = strings.TrimSpace(req.Note)
	return nil
}

// toEntry builds an Entry from an already-validated request.
func toEntry(req *createRequest) *Entry {
	return &Entry{
		PlanDate: req.planDate,
		MealType: req.MealType,
		RecipeID: req.RecipeID,
		Note:     req.Note,
	}
}

// List serves GET /api/meal-plan with optional date_from / date_to filters.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, ok := optionalDate(w, q.Get("date_from"), "date_from")
	if !ok {
		return
	}
	to, ok := optionalDate(w, q.Get("date_to"), "date_to")
	if !ok {
		return
	}
	rows, err := h.store.List(r.Context(), from, to)
	if err != nil {
		h.logger.Error("list meal plan", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]response, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Create serves POST /api/meal-plan.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httputil.DecodeJSON(w, r, 1<<16, &req) {
		return
	}
	if vErr := validate(&req); vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	created, err := h.store.Create(r.Context(), toEntry(&req))
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

// Update serves PUT /api/meal-plan/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "entry_id")
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
	updated, err := h.store.Update(r.Context(), id, toEntry(&req))
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(updated))
}

// Delete serves DELETE /api/meal-plan/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "entry_id")
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Meal plan entry not found")
			return
		}
		h.logger.Error("delete meal plan entry", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error) {
	var missing *RecipeMissingError
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound, "Meal plan entry not found")
	case errors.As(err, &missing):
		httputil.WriteDetailError(w, http.StatusNotFound, missing.Error())
	default:
		h.logger.Error("meal plan store", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

// optionalDate parses an optional YYYY-MM-DD query value. Empty → (nil, true).
func optionalDate(w http.ResponseWriter, raw, field string) (*time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, true
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		httputil.WriteBodyValidationError(w, field, "must be YYYY-MM-DD", raw)
		return nil, false
	}
	return &t, true
}
