package kafka

import (
	"context"
	"errors"
	"testing"
	"vehicle-service/internal/domain/events"
	"vehicle-service/internal/domain/ports"
)

// recordingService implements only RecordServiceCompleted; the embedded
// nil interface makes any other call panic, which the test would catch.
type recordingService struct {
	ports.VehicleService
	got events.ServiceCompleted
	err error
}

func (s *recordingService) RecordServiceCompleted(_ context.Context, e events.ServiceCompleted) error {
	s.got = e
	return s.err
}

func TestServiceCompletedHandler(t *testing.T) {
	valid := []byte(`{"event_id":"e1","appointment_id":500,"vehicle_id":1,"dealership_id":2,
		"completed_at":"2026-09-01T14:00:00Z","services":[{"service_id":3,"name":"Oil change"}]}`)

	t.Run("decodes and records", func(t *testing.T) {
		svc := &recordingService{}
		if err := ServiceCompletedHandler(svc)(context.Background(), []byte("500"), valid); err != nil {
			t.Fatal(err)
		}
		if svc.got.AppointmentID != 500 || len(svc.got.Services) != 1 || svc.got.Services[0].Name != "Oil change" {
			t.Errorf("recorded %+v", svc.got)
		}
	})

	// These can never succeed on retry, so they're skipped (nil) rather
	// than left to block the partition.
	for name, tc := range map[string]struct {
		value []byte
		err   error
	}{
		"undecodable":     {[]byte(`{not json`), nil},
		"invalid event":   {valid, ports.ErrInvalidInput},
		"unknown vehicle": {valid, ports.ErrNotFound},
	} {
		t.Run("skips "+name, func(t *testing.T) {
			if err := ServiceCompletedHandler(&recordingService{err: tc.err})(context.Background(), nil, tc.value); err != nil {
				t.Errorf("err = %v, want nil (skip)", err)
			}
		})
	}

	t.Run("returns other errors", func(t *testing.T) {
		boom := errors.New("db down")
		if err := ServiceCompletedHandler(&recordingService{err: boom})(context.Background(), nil, valid); !errors.Is(err, boom) {
			t.Errorf("err = %v, want %v", err, boom)
		}
	})
}
