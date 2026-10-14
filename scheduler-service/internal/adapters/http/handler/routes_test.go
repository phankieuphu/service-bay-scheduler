package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"scheduler-service/internal/adapters/http/middleware"
	"scheduler-service/internal/domain/entity"
	"scheduler-service/internal/domain/ports"
	"strings"
	"testing"

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
type panicService struct{ ports.BookingService }

func TestRoutes_Roles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	NewBookingHandler(panicService{}).RegisterRoutes(r.Group("/api/v1", middleware.RequireAuth(roleTokens{})))

	routes := []struct {
		method, path, body string
		customerOnly       bool
	}{
		{http.MethodGet, "/api/v1/hubs", "", false},
		{http.MethodGet, "/api/v1/services", "", false},
		{http.MethodGet, "/api/v1/hubs/1/availability?date=2026-10-20&service_id=1", "", false},
		{http.MethodPost, "/api/v1/appointments", `{"hub_id":1,"vehicle_id":1,"service_ids":[1],"start":"2026-10-20T10:00:00+07:00"}`, true},
		{http.MethodGet, "/api/v1/me/appointments", "", true},
		{http.MethodPost, "/api/v1/appointments/1/cancel", "", true},
	}

	for _, rt := range routes {
		for _, token := range []string{"", "bogus", "CUSTOMER", "TECHNICIAN", "ADMIN"} {
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
				case rt.customerOnly && token != "CUSTOMER":
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
