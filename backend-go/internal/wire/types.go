// Package wire holds JSON wire types shared across HTTP handlers: an ISO
// date-only value and a timezone-less ISO timestamp, so handlers serialize
// dates and timestamps consistently across domains.
package wire

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	isoDateLayout  = "2006-01-02"
	isoNaiveLayout = "2006-01-02T15:04:05.999999"
)

// IsoDate marshals a time.Time as a date-only YYYY-MM-DD string.
type IsoDate time.Time

func (d IsoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format(isoDateLayout) + `"`), nil
}

func (d *IsoDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("expected YYYY-MM-DD string: %w", err)
	}
	t, err := time.Parse(isoDateLayout, s)
	if err != nil {
		return fmt.Errorf("expected YYYY-MM-DD: %w", err)
	}
	*d = IsoDate(t)
	return nil
}

// IsoNaive marshals a time.Time as an ISO-8601 timestamp without a zone
// suffix.
type IsoNaive time.Time

func (t IsoNaive) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format(isoNaiveLayout) + `"`), nil
}
