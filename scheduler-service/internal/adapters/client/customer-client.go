package client

import (
	"context"
	"net/http"
	"scheduler-service/internal/domain/entity"
	"scheduler-service/internal/domain/ports"
	"strings"
	"time"
)

// CustomerClient implements [ports.CustomerDirectory] over customer-service's
// GET /customer/me.
type CustomerClient struct {
	baseURL string
	http    *http.Client
}

func NewCustomerClient(baseURL string, timeout time.Duration) *CustomerClient {
	return &CustomerClient{baseURL: strings.TrimRight(baseURL, "/"), http: newHTTPClient(timeout)}
}

type customerResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CurrentCustomer implements [ports.CustomerDirectory].
func (c *CustomerClient) CurrentCustomer(ctx context.Context, token string) (entity.Customer, error) {
	var body customerResponse
	status, err := getJSON(ctx, c.http, c.baseURL+"/customer/me", token, &body)
	if status == http.StatusNotFound {
		return entity.Customer{}, ports.ErrNoCustomerProfile
	}
	if err != nil {
		return entity.Customer{}, err
	}
	return entity.Customer{ID: body.ID, Name: body.Name, Email: body.Email}, nil
}
