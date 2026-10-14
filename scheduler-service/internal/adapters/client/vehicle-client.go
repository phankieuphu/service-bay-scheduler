package client

import (
	"context"
	"net/http"
	"scheduler-service/internal/domain/entity"
	"strings"
	"time"
)

// VehicleClient implements [ports.VehicleDirectory] over vehicle-service's
// GET /customers/:id/vehicles, which lists the vehicles a customer owns now.
type VehicleClient struct {
	baseURL string
	http    *http.Client
}

func NewVehicleClient(baseURL string, timeout time.Duration) *VehicleClient {
	return &VehicleClient{baseURL: strings.TrimRight(baseURL, "/"), http: newHTTPClient(timeout)}
}

type customerVehiclesResponse struct {
	Vehicles []struct {
		ID           int64  `json:"id"`
		VIN          string `json:"vin"`
		LicensePlate string `json:"license_plate"`
	} `json:"vehicles"`
}

// CustomerVehicles implements [ports.VehicleDirectory].
func (c *VehicleClient) CustomerVehicles(ctx context.Context, token string, customerID int64) ([]entity.Vehicle, error) {
	var body customerVehiclesResponse
	if _, err := getJSON(ctx, c.http, c.baseURL+"/customers/"+itoa(customerID)+"/vehicles", token, &body); err != nil {
		return nil, err
	}

	vehicles := make([]entity.Vehicle, len(body.Vehicles))
	for i, v := range body.Vehicles {
		vehicles[i] = entity.Vehicle{ID: v.ID, VIN: v.VIN, LicensePlate: v.LicensePlate}
	}
	return vehicles, nil
}
