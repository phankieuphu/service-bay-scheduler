package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"vehicle-service/internal/domain/ports"
)

// vinPattern is ISO 3779: 17 characters, digits and capital letters except
// I, O and Q (too easily confused with 1 and 0).
var vinPattern = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)

const maxLicensePlateLength = 20

// normalizeVin uppercases and trims vin, and rejects anything that isn't a
// valid VIN.
func normalizeVin(vin string) (string, error) {
	vin = strings.ToUpper(strings.TrimSpace(vin))
	if !vinPattern.MatchString(vin) {
		return "", fmt.Errorf("%w: vin must be 17 characters of A-Z (except I, O, Q) and 0-9", ports.ErrInvalidInput)
	}
	return vin, nil
}

// normalizePlate uppercases plate and drops everything but A-Z and 0-9, so
// "51a-123.45" and "51A 12345" are the same plate. "" means no plate.
func normalizePlate(plate string) (string, error) {
	var b strings.Builder
	for _, r := range strings.ToUpper(plate) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	normalized := b.String()
	if len(normalized) > maxLicensePlateLength {
		return "", fmt.Errorf("%w: license_plate is longer than %d characters", ports.ErrInvalidInput, maxLicensePlateLength)
	}
	if normalized == "" && strings.TrimSpace(plate) != "" {
		return "", fmt.Errorf("%w: license_plate has no letters or digits", ports.ErrInvalidInput)
	}
	return normalized, nil
}

// calendarDate keeps only the date part of t, read in t's own offset, as
// midnight UTC. A client sending local midnight ("2030-07-11T00:00:00+07:00")
// means July 11; converting to UTC first would turn it into July 10.
func calendarDate(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}

func sameDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
