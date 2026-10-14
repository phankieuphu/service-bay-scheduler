package times

import "time"

// WireLayout is the format for every timestamp in HTTP and Kafka payloads:
// UTC, fixed millisecond precision, e.g. 2026-09-24T08:15:30.123Z.
const WireLayout = "2006-01-02T15:04:05.000Z07:00"

// Time wraps time.Time so it always marshals in WireLayout. Use it only in
// DTOs and event payloads; domain entities keep plain time.Time.
type Time struct{ time.Time }

func NewTime(t time.Time) Time {
	return Time{t.UTC().Truncate(time.Millisecond)}
}

func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + t.UTC().Format(WireLayout) + `"`), nil
}

// UnmarshalJSON accepts any RFC3339 input (any fractional precision or zone
// offset) and normalizes it to UTC.
func (t *Time) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*t = Time{}
		return nil
	}
	parsed, err := time.Parse(`"`+time.RFC3339Nano+`"`, string(b))
	if err != nil {
		return err
	}
	*t = NewTime(parsed)
	return nil
}
