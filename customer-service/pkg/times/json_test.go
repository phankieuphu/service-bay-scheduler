package times

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTime_MarshalJSON(t *testing.T) {
	utc7 := time.FixedZone("UTC+7", 7*60*60)

	tests := []struct {
		name string
		in   Time
		want string
	}{
		{"pads to 3 digits", Time{time.Date(2026, 9, 24, 8, 15, 30, 120_000_000, time.UTC)}, `"2026-09-24T08:15:30.120Z"`},
		{"truncates nanoseconds", Time{time.Date(2026, 9, 24, 8, 15, 30, 123_456_789, time.UTC)}, `"2026-09-24T08:15:30.123Z"`},
		{"converts to UTC", Time{time.Date(2026, 9, 24, 15, 15, 30, 0, utc7)}, `"2026-09-24T08:15:30.000Z"`},
		{"zero is null", Time{}, `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTime_UnmarshalJSON(t *testing.T) {
	var got Time
	if err := json.Unmarshal([]byte(`"2026-09-24T15:15:30.123456+07:00"`), &got); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 9, 24, 8, 15, 30, 123_000_000, time.UTC)
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("got %v, want %v", got.Time, want)
	}
}
