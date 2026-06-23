package nutrition

import (
	"sort"
	"time"
)

// Macros is the set of four tracked nutrition values.
type Macros struct {
	CaloriesKcal float64
	ProteinG     float64
	CarbsG       float64
	FatG         float64
}

// add returns the element-wise sum of two macro sets.
func (m Macros) add(o Macros) Macros {
	return Macros{
		CaloriesKcal: m.CaloriesKcal + o.CaloriesKcal,
		ProteinG:     m.ProteinG + o.ProteinG,
		CarbsG:       m.CarbsG + o.CarbsG,
		FatG:         m.FatG + o.FatG,
	}
}

// sub returns m minus o (positive means m exceeds o).
func (m Macros) sub(o Macros) Macros {
	return Macros{
		CaloriesKcal: m.CaloriesKcal - o.CaloriesKcal,
		ProteinG:     m.ProteinG - o.ProteinG,
		CarbsG:       m.CarbsG - o.CarbsG,
		FatG:         m.FatG - o.FatG,
	}
}

// scale divides each macro by n (n must be > 0).
func (m Macros) scale(n float64) Macros {
	return Macros{
		CaloriesKcal: m.CaloriesKcal / n,
		ProteinG:     m.ProteinG / n,
		CarbsG:       m.CarbsG / n,
		FatG:         m.FatG / n,
	}
}

// DaySummary is one day's macro total plus the number of planned meals.
type DaySummary struct {
	Date  time.Time
	Total Macros
	Meals int
}

// Summary is the aggregated result over the requested range: per-day totals,
// the period total, the per-day average (over days that have meals) and the
// optional daily targets the caller supplied.
type Summary struct {
	Days    []DaySummary
	Totals  Macros
	Meals   int
	Average Macros
	Targets *Macros
}

// summarize groups contributions by calendar day, then computes per-day totals,
// the period total and the per-day average over days that have planned meals.
// Targets, if any, are passed through unchanged for the handler to diff per day.
func summarize(contribs []Contribution, targets *Macros) Summary {
	byDay := map[string]DaySummary{}
	keys := []string{}
	for i := range contribs {
		c := contribs[i]
		key := c.Date.Format("2006-01-02")
		d, ok := byDay[key]
		if !ok {
			d = DaySummary{Date: c.Date}
			keys = append(keys, key)
		}
		d.Total = d.Total.add(Macros{c.CaloriesKcal, c.ProteinG, c.CarbsG, c.FatG})
		d.Meals++
		byDay[key] = d
	}
	sort.Strings(keys) // YYYY-MM-DD sorts chronologically

	days := make([]DaySummary, 0, len(keys))
	var totals Macros
	meals := 0
	for _, k := range keys {
		d := byDay[k]
		days = append(days, d)
		totals = totals.add(d.Total)
		meals += d.Meals
	}

	var average Macros
	if len(days) > 0 {
		average = totals.scale(float64(len(days)))
	}

	return Summary{Days: days, Totals: totals, Meals: meals, Average: average, Targets: targets}
}
