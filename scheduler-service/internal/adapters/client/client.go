// Package client calls customer-service and vehicle-service on behalf of
// the signed-in caller, forwarding their access token. These are the
// synchronous write-time checks of architecture-design.md §4 option 1.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"scheduler-service/internal/domain/ports"
	"strconv"
	"time"
)

// maxErrorBody bounds how much of an upstream error response is read into
// the error message.
const maxErrorBody = 512

// getJSON GETs url with the caller's token and decodes a 200 response into
// out. It returns the status code so callers can map expected ones (404);
// transport failures and other non-200s are wrapped in ports.ErrUpstream.
func getJSON(ctx context.Context, httpClient *http.Client, url, token string, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ports.ErrUpstream, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return resp.StatusCode, fmt.Errorf("%w: GET %s returned %d: %s", ports.ErrUpstream, req.URL.Path, resp.StatusCode, body)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return resp.StatusCode, fmt.Errorf("%w: decode %s: %v", ports.ErrUpstream, req.URL.Path, err)
	}
	return resp.StatusCode, nil
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }
