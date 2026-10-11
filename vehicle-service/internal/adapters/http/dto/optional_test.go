package dto

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUpdateVehicleDTO_WarrantyEndDatePresence(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantSet   bool
		wantValue bool
	}{
		{"omitted leaves it unchanged", `{"updated_at":"2026-09-01T10:00:00Z"}`, false, false},
		{"null clears it", `{"warranty_end_date":null,"updated_at":"2026-09-01T10:00:00Z"}`, true, false},
		{"a date sets it", `{"warranty_end_date":"2029-06-30T00:00:00Z","updated_at":"2026-09-01T10:00:00Z"}`, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req UpdateVehicleDTO
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatal(err)
			}
			if req.WarrantyEndDate.Set != tt.wantSet || (req.WarrantyEndDate.Value != nil) != tt.wantValue {
				t.Errorf("got Set=%v Value=%v, want Set=%v hasValue=%v",
					req.WarrantyEndDate.Set, req.WarrantyEndDate.Value, tt.wantSet, tt.wantValue)
			}
			if tt.wantValue && !req.WarrantyEndDate.Value.Equal(time.Date(2029, 6, 30, 0, 0, 0, 0, time.UTC)) {
				t.Errorf("value = %v", req.WarrantyEndDate.Value)
			}
		})
	}
}

func TestOptional_RejectsBadValue(t *testing.T) {
	var req UpdateVehicleDTO
	if err := json.Unmarshal([]byte(`{"warranty_end_date":"not a date"}`), &req); err == nil {
		t.Error("expected an error for a malformed date")
	}
}
