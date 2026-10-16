package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"vehicle-service/internal/adapters/http/middleware"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"

	"github.com/gin-gonic/gin"
)

// roleTokens accepts a token named after a role ("CUSTOMER", "ADMIN", ...).
type roleTokens struct{}

func (roleTokens) ParseAccessToken(token string) (entity.Principal, error) {
	switch entity.Role(token) {
	case entity.RoleCustomer, entity.RoleTechnician, entity.RoleManager, entity.RoleAdmin:
		return entity.Principal{UserID: "u-" + token, Role: entity.Role(token)}, nil
	}
	return entity.Principal{}, errors.New("invalid token")
}

// panicService has no methods: a request that gets past the auth checks
// panics into gin.Recovery's 500, which is how the test tells it got through.
type panicService struct{ ports.VehicleService }

func TestRoutes_ReadsAuthenticatedWritesStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	NewVehicleHandler(panicService{}).RegisterRoutes(r.Group("/api/v1", middleware.RequireAuth(roleTokens{})))

	routes := []struct {
		method, path, body string
		write              bool
	}{
		{http.MethodGet, "/api/v1/customers/1/vehicles", "", false},
		{http.MethodGet, "/api/v1/vehicle", "", false},
		{http.MethodGet, "/api/v1/vehicle/1", "", false},
		{http.MethodGet, "/api/v1/vehicle/1/warranty", "", false},
		{http.MethodGet, "/api/v1/vehicle/1/history", "", false},
		{http.MethodGet, "/api/v1/vehicle/1/materials", "", false},
		{http.MethodPost, "/api/v1/transfer", `{"vehicle_id":1,"from":1,"to":2}`, true},
		{http.MethodPost, "/api/v1/vehicle", `{"vin":"1HGCM82633A004352","vehicle_model_id":1}`, true},
		{http.MethodPatch, "/api/v1/vehicle/1", `{"status":"SOLD","updated_at":"2026-01-01T00:00:00Z"}`, true},
		{http.MethodGet, "/api/v1/vehicle/1/owner", "", true}, // staff-only read: exposes a customer id
		{http.MethodPost, "/api/v1/vehicle/1/owner", `{"customer_id":1}`, true},
	}

	for _, rt := range routes {
		for _, token := range []string{"", "bogus", "CUSTOMER", "TECHNICIAN", "MANAGER", "ADMIN"} {
			t.Run(rt.method+" "+rt.path+" as "+token, func(t *testing.T) {
				req := httptest.NewRequest(rt.method, rt.path, strings.NewReader(rt.body))
				req.Header.Set("Content-Type", "application/json")
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				switch {
				case token == "" || token == "bogus":
					if w.Code != http.StatusUnauthorized {
						t.Errorf("code = %d, want 401", w.Code)
					}
				case rt.write && token == "CUSTOMER":
					if w.Code != http.StatusForbidden {
						t.Errorf("code = %d, want 403", w.Code)
					}
				default:
					if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
						t.Errorf("code = %d, want the request to reach the handler", w.Code)
					}
				}
			})
		}
	}
}
