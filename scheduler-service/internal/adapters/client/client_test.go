package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"scheduler-service/internal/domain/ports"
	"testing"
	"time"
)

func TestCustomerClient_ForwardsTokenAndMaps404(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/api/v1/customer/me":
			if r.Header.Get("Authorization") == "Bearer linked" {
				w.Write([]byte(`{"id":42,"name":"Jane","email":"jane@example.com"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"no customer profile is linked to this account"}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	c := NewCustomerClient(srv.URL+"/api/v1/", time.Second)

	customer, err := c.CurrentCustomer(t.Context(), "linked")
	if err != nil || customer.ID != 42 || customer.Email != "jane@example.com" {
		t.Fatalf("CurrentCustomer = %+v, %v", customer, err)
	}
	if gotAuth != "Bearer linked" {
		t.Errorf("Authorization = %q", gotAuth)
	}

	if _, err := c.CurrentCustomer(t.Context(), "unlinked"); !errors.Is(err, ports.ErrNoCustomerProfile) {
		t.Errorf("err = %v, want ErrNoCustomerProfile", err)
	}
}

func TestVehicleClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/customers/42/vehicles" {
			w.Write([]byte(`{"vehicles":[{"id":7,"vin":"VIN7","license_plate":"51A-123.45","owned_from":"2026-01-01"}]}`))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	c := NewVehicleClient(srv.URL, time.Second)

	vehicles, err := c.CustomerVehicles(t.Context(), "t", 42)
	if err != nil || len(vehicles) != 1 || vehicles[0].ID != 7 || vehicles[0].LicensePlate != "51A-123.45" {
		t.Fatalf("CustomerVehicles = %+v, %v", vehicles, err)
	}
	if _, err := c.CustomerVehicles(t.Context(), "t", 1); !errors.Is(err, ports.ErrUpstream) {
		t.Errorf("err = %v, want ErrUpstream", err)
	}
}
