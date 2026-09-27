package services

import (
	"context"
	"errors"
	"testing"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/ports"
)

func TestVehicleService_ListVehicles(t *testing.T) {
	newSvc := func(list func(context.Context, ports.ListVehiclesParams) (ports.VehiclePage, error)) ports.VehicleService {
		return NewVehicleService(config.Config{}, &MockVehicleRepository{ListFunc: list}, &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{})
	}

	limits := []struct {
		in, want int
	}{{0, defaultListLimit}, {-5, defaultListLimit}, {50, 50}, {1000, maxListLimit}}
	for _, tt := range limits {
		var got ports.ListVehiclesParams
		svc := newSvc(func(_ context.Context, p ports.ListVehiclesParams) (ports.VehiclePage, error) {
			got = p
			return ports.VehiclePage{}, nil
		})
		if _, err := svc.ListVehicles(context.Background(), ports.ListVehiclesParams{Limit: tt.in}); err != nil {
			t.Fatal(err)
		}
		if got.Limit != tt.want {
			t.Errorf("limit %d -> %d, want %d", tt.in, got.Limit, tt.want)
		}
	}

	t.Run("search terms are normalized like on write", func(t *testing.T) {
		var got ports.ListVehiclesParams
		svc := newSvc(func(_ context.Context, p ports.ListVehiclesParams) (ports.VehiclePage, error) {
			got = p
			return ports.VehiclePage{}, nil
		})
		_, err := svc.ListVehicles(context.Background(), ports.ListVehiclesParams{
			Vin: " 1hgcm82633a004352", LicensePlate: "51a-123.45", Status: constants.StatusSold,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Vin != validVin || got.LicensePlate != "51A12345" || got.Status != constants.StatusSold {
			t.Errorf("repo got %+v", got)
		}
	})

	for name, params := range map[string]ports.ListVehiclesParams{
		"malformed vin":  {Vin: "123"},
		"unknown status": {Status: "STOLEN"},
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			svc := newSvc(func(context.Context, ports.ListVehiclesParams) (ports.VehiclePage, error) {
				t.Error("repository called")
				return ports.VehiclePage{}, nil
			})
			if _, err := svc.ListVehicles(context.Background(), params); !errors.Is(err, ports.ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
}
