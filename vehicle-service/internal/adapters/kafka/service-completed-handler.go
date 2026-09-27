package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"vehicle-service/internal/domain/events"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"
)

// ServiceCompletedHandler feeds scheduler-service's ServiceCompleted events
// into the vehicle's service history.
//
// A message that can never succeed — undecodable, missing fields, or for a
// vehicle this service doesn't know — is logged and skipped (nil), since
// retrying it would only block the partition. Any other error is returned.
func ServiceCompletedHandler(service ports.VehicleService) func(ctx context.Context, key, value []byte) error {
	return func(ctx context.Context, key, value []byte) error {
		var event events.ServiceCompleted
		if err := json.Unmarshal(value, &event); err != nil {
			logger.ErrorContext(ctx, "ServiceCompleted: undecodable message skipped", "key", string(key), "error", err)
			return nil
		}

		err := service.RecordServiceCompleted(ctx, event)
		switch {
		case errors.Is(err, ports.ErrInvalidInput), errors.Is(err, ports.ErrNotFound):
			logger.ErrorContext(ctx, "ServiceCompleted: message skipped", "appointment", event.AppointmentID, "vehicle", event.VehicleID, "error", err)
			return nil
		case err != nil:
			return err
		}
		return nil
	}
}
