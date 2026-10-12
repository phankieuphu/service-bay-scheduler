package handler

import (
	"customer-service/internal/adapters/http/middleware"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
	"errors"
	"net/http"
	"net/http/httptest"
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
type panicService struct{ ports.CustomerService }

func TestRoutes_RequireStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	NewCustomerHandler(panicService{}).RegisterRoutes(r.Group("/api/v1", middleware.RequireAuth(roleTokens{})))

	routes := []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/customer", `{"name":"Jane","email":"jane@example.com","birth_day":"1990-01-01T00:00:00Z"}`},
		{http.MethodGet, "/api/v1/customer", ""},
		{http.MethodGet, "/api/v1/customer/1", ""},
		{http.MethodPut, "/api/v1/customer/1", `{"name":"Jane","updated_at":"2026-01-01T00:00:00Z"}`},
		{http.MethodDelete, "/api/v1/customer/1", ""},
	}
	allowed := func(method string, role entity.Role) bool {
		switch role {
		case entity.RoleManager, entity.RoleAdmin:
			return true
		case entity.RoleTechnician:
			return method != http.MethodDelete
		}
		return false
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
				case allowed(rt.method, entity.Role(token)):
					if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
						t.Errorf("code = %d, want the request to reach the handler", w.Code)
					}
				default:
					if w.Code != http.StatusForbidden {
						t.Errorf("code = %d, want 403", w.Code)
					}
				}
			})
		}
	}
}
