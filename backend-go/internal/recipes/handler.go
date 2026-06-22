package recipes

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
	"github.com/Automaat/baratie/backend-go/internal/wire"
)

// response is the JSON shape returned for a recipe.
type response struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Instructions string        `json:"instructions"`
	Ingredients  []string      `json:"ingredients"`
	Tags         []string      `json:"tags"`
	Servings     int           `json:"servings"`
	PrepMinutes  int           `json:"prep_minutes"`
	CookMinutes  int           `json:"cook_minutes"`
	TotalMinutes int           `json:"total_minutes"`
	CreatedAt    wire.IsoNaive `json:"created_at"`
}

// createRequest is the body accepted by POST and PUT.
type createRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Instructions string   `json:"instructions"`
	Ingredients  []string `json:"ingredients"`
	Tags         []string `json:"tags"`
	Servings     int      `json:"servings"`
	PrepMinutes  int      `json:"prep_minutes"`
	CookMinutes  int      `json:"cook_minutes"`
}

// Handler is the HTTP boundary for /api/recipes.
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

func toResponse(r *Recipe) response {
	return response{
		ID:           r.ID,
		Name:         r.Name,
		Description:  r.Description,
		Instructions: r.Instructions,
		Ingredients:  r.Ingredients,
		Tags:         r.Tags,
		Servings:     r.Servings,
		PrepMinutes:  r.PrepMinutes,
		CookMinutes:  r.CookMinutes,
		TotalMinutes: r.PrepMinutes + r.CookMinutes,
		CreatedAt:    wire.IsoNaive(r.CreatedAt),
	}
}

func toRecipe(req *createRequest) *Recipe {
	return &Recipe{
		Name:         req.Name,
		Description:  req.Description,
		Instructions: req.Instructions,
		Ingredients:  req.Ingredients,
		Tags:         req.Tags,
		Servings:     req.Servings,
		PrepMinutes:  req.PrepMinutes,
		CookMinutes:  req.CookMinutes,
	}
}

// List serves GET /api/recipes.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list recipes", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]response, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Get serves GET /api/recipes/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "recipe_id")
	if !ok {
		return
	}
	recipe, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Recipe not found")
			return
		}
		h.logger.Error("get recipe", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(recipe))
}

// Create serves POST /api/recipes.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httputil.DecodeJSON(w, r, 1<<18, &req) {
		return
	}
	if vErr := validate(&req); vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	created, err := h.store.Create(r.Context(), toRecipe(&req))
	if err != nil {
		h.logger.Error("create recipe", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

// Update serves PUT /api/recipes/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "recipe_id")
	if !ok {
		return
	}
	var req createRequest
	if !httputil.DecodeJSON(w, r, 1<<18, &req) {
		return
	}
	if vErr := validate(&req); vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	updated, err := h.store.Update(r.Context(), id, toRecipe(&req))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Recipe not found")
			return
		}
		h.logger.Error("update recipe", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(updated))
}

// Delete serves DELETE /api/recipes/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "recipe_id")
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Recipe not found")
			return
		}
		h.logger.Error("delete recipe", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
