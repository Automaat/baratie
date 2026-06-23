package nutrition

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Automaat/baratie/backend-go/internal/httputil"
	"github.com/Automaat/baratie/backend-go/internal/wire"
)

// macrosResponse is the JSON shape for a set of macros (totals, average,
// targets, per-day deltas all reuse it).
type macrosResponse struct {
	CaloriesKcal float64 `json:"calories_kcal"`
	ProteinG     float64 `json:"protein_g"`
	CarbsG       float64 `json:"carbs_g"`
	FatG         float64 `json:"fat_g"`
}

func macrosToResponse(m Macros) macrosResponse {
	return macrosResponse(m)
}

// dayResponse is one day's totals plus the meal count and, when daily targets
// were supplied, that day's signed delta vs target (positive = over target).
type dayResponse struct {
	Date         wire.IsoDate    `json:"date"`
	CaloriesKcal float64         `json:"calories_kcal"`
	ProteinG     float64         `json:"protein_g"`
	CarbsG       float64         `json:"carbs_g"`
	FatG         float64         `json:"fat_g"`
	Meals        int             `json:"meals"`
	TargetDelta  *macrosResponse `json:"target_delta"`
}

// totalsResponse is the period total plus meal and day counts.
type totalsResponse struct {
	CaloriesKcal float64 `json:"calories_kcal"`
	ProteinG     float64 `json:"protein_g"`
	CarbsG       float64 `json:"carbs_g"`
	FatG         float64 `json:"fat_g"`
	Meals        int     `json:"meals"`
	Days         int     `json:"days"`
}

// summaryResponse is the full body returned by GET /api/nutrition/summary.
type summaryResponse struct {
	Days    []dayResponse   `json:"days"`
	Totals  totalsResponse  `json:"totals"`
	Average macrosResponse  `json:"average"`
	Targets *macrosResponse `json:"targets"`
}

// Handler is the HTTP boundary for /api/nutrition.
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

// Summary serves GET /api/nutrition/summary with optional date_from / date_to
// filters and optional target_kcal / target_protein_g / target_carbs_g /
// target_fat_g daily targets.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, ok := httputil.OptionalDate(w, q.Get("date_from"), "date_from")
	if !ok {
		return
	}
	to, ok := httputil.OptionalDate(w, q.Get("date_to"), "date_to")
	if !ok {
		return
	}
	targets, hasTargets, vErr := parseTargets(q)
	if vErr != nil {
		httputil.WriteValidationError(w, vErr)
		return
	}
	var targetsPtr *Macros
	if hasTargets {
		targetsPtr = &targets
	}

	contribs, err := h.store.Contributions(r.Context(), from, to)
	if err != nil {
		h.logger.Error("nutrition summary", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(summarize(contribs, targetsPtr)))
}

// toResponse maps the aggregated Summary onto the JSON wire shape, attaching a
// per-day delta vs target when daily targets were supplied.
func toResponse(s Summary) summaryResponse {
	days := make([]dayResponse, 0, len(s.Days))
	for _, d := range s.Days {
		day := dayResponse{
			Date:         wire.IsoDate(d.Date),
			CaloriesKcal: d.Total.CaloriesKcal,
			ProteinG:     d.Total.ProteinG,
			CarbsG:       d.Total.CarbsG,
			FatG:         d.Total.FatG,
			Meals:        d.Meals,
		}
		if s.Targets != nil {
			delta := macrosToResponse(d.Total.sub(*s.Targets))
			day.TargetDelta = &delta
		}
		days = append(days, day)
	}

	resp := summaryResponse{
		Days: days,
		Totals: totalsResponse{
			CaloriesKcal: s.Totals.CaloriesKcal,
			ProteinG:     s.Totals.ProteinG,
			CarbsG:       s.Totals.CarbsG,
			FatG:         s.Totals.FatG,
			Meals:        s.Meals,
			Days:         len(s.Days),
		},
		Average: macrosToResponse(s.Average),
	}
	if s.Targets != nil {
		t := macrosToResponse(*s.Targets)
		resp.Targets = &t
	}
	return resp
}

// parseTargets reads the optional daily-target query params. The bool reports
// whether any target was supplied; when true, the missing ones default to 0. A
// malformed or negative value is a 422-shaped ValidationError.
func parseTargets(q url.Values) (Macros, bool, *httputil.ValidationError) {
	var t Macros
	fields := []struct {
		key string
		dst *float64
	}{
		{"target_kcal", &t.CaloriesKcal},
		{"target_protein_g", &t.ProteinG},
		{"target_carbs_g", &t.CarbsG},
		{"target_fat_g", &t.FatG},
	}
	present := false
	for _, f := range fields {
		raw := strings.TrimSpace(q.Get(f.key))
		if raw == "" {
			continue
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v < 0 {
			return Macros{}, false, &httputil.ValidationError{Field: f.key, Msg: "must be a non-negative number"}
		}
		*f.dst = v
		present = true
	}
	return t, present, nil
}
