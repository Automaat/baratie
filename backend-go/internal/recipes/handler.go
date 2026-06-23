package recipes

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
	"github.com/Automaat/baratie/backend-go/internal/units"
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
	CaloriesKcal float64       `json:"calories_kcal"`
	ProteinG     float64       `json:"protein_g"`
	CarbsG       float64       `json:"carbs_g"`
	FatG         float64       `json:"fat_g"`
	Structured   []ingredient  `json:"ingredients_structured"`
	CreatedAt    wire.IsoNaive `json:"created_at"`
}

// ingredient is the JSON shape for a structured (food-linked) ingredient.
type ingredient struct {
	ID             int     `json:"id"`
	FoodID         int     `json:"food_id"`
	FoodName       string  `json:"food_name"`
	Amount         float64 `json:"amount"`
	Unit           string  `json:"unit"`
	KcalPer100g    float64 `json:"kcal_per_100g"`
	ProteinPer100g float64 `json:"protein_per_100g"`
	CarbsPer100g   float64 `json:"carbs_per_100g"`
	FatPer100g     float64 `json:"fat_per_100g"`
}

// ingredientsRequest is the body accepted by PUT /api/recipes/{id}/ingredients.
type ingredientsRequest struct {
	Ingredients []ingredientInput `json:"ingredients"`
}

// ingredientInput is one structured ingredient on the write path.
type ingredientInput struct {
	FoodID int     `json:"food_id"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
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
	CaloriesKcal float64  `json:"calories_kcal"`
	ProteinG     float64  `json:"protein_g"`
	CarbsG       float64  `json:"carbs_g"`
	FatG         float64  `json:"fat_g"`
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
		CaloriesKcal: r.CaloriesKcal,
		ProteinG:     r.ProteinG,
		CarbsG:       r.CarbsG,
		FatG:         r.FatG,
		Structured:   toIngredients(r.Structured),
		CreatedAt:    wire.IsoNaive(r.CreatedAt),
	}
}

func toIngredients(in []StructuredIngredient) []ingredient {
	out := make([]ingredient, 0, len(in))
	for _, si := range in {
		out = append(out, ingredient{
			ID:             si.ID,
			FoodID:         si.FoodID,
			FoodName:       si.FoodName,
			Amount:         si.Amount,
			Unit:           si.Unit,
			KcalPer100g:    si.KcalPer100g,
			ProteinPer100g: si.ProteinPer100g,
			CarbsPer100g:   si.CarbsPer100g,
			FatPer100g:     si.FatPer100g,
		})
	}
	return out
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
		CaloriesKcal: req.CaloriesKcal,
		ProteinG:     req.ProteinG,
		CarbsG:       req.CarbsG,
		FatG:         req.FatG,
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

// ReplaceIngredients serves PUT /api/recipes/{id}/ingredients — a full replace
// of the recipe's structured (food-linked) ingredients.
func (h *Handler) ReplaceIngredients(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.PathIntField(w, r, "id", "recipe_id")
	if !ok {
		return
	}
	var req ingredientsRequest
	if !httputil.DecodeJSON(w, r, 1<<18, &req) {
		return
	}
	inputs, vErr := validateIngredients(req.Ingredients)
	if vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	updated, err := h.store.ReplaceIngredients(r.Context(), id, inputs)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteDetailError(w, http.StatusNotFound, "Recipe not found")
		case errors.Is(err, ErrFoodMissing):
			httputil.WriteBodyValidationError(w, "food_id", "references a food that does not exist", "")
		default:
			h.logger.Error("replace recipe ingredients", "err", err)
			httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(updated))
}

// validateIngredients normalizes the inputs (defaulting blank units to grams)
// and rejects bad food ids, negative amounts and unknown units.
func validateIngredients(in []ingredientInput) ([]IngredientInput, *httputil.ValidationError) {
	out := make([]IngredientInput, 0, len(in))
	for _, item := range in {
		if item.FoodID <= 0 {
			return nil, &httputil.ValidationError{Field: "food_id", Msg: "must be a positive food id"}
		}
		if item.Amount < 0 {
			return nil, &httputil.ValidationError{Field: "amount", Msg: "Amount cannot be negative"}
		}
		unit := strings.ToLower(strings.TrimSpace(item.Unit))
		if unit == "" {
			unit = units.Gram
		}
		if !units.Known(unit) {
			return nil, &httputil.ValidationError{Field: "unit", Msg: "unknown unit"}
		}
		out = append(out, IngredientInput{FoodID: item.FoodID, Amount: item.Amount, Unit: unit})
	}
	return out, nil
}
