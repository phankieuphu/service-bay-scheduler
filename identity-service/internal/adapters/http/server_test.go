package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware([]string{"http://localhost:5173"}))
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	preflight := func(origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	w := preflight("http://localhost:5173")
	if w.Code != http.StatusNoContent ||
		w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" ||
		w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("listed origin: code %d, headers %v", w.Code, w.Header())
	}

	w = preflight("https://evil.example")
	if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Errorf("unlisted origin got CORS headers: %v", w.Header())
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Error("responses must Vary on Origin, since the CORS headers depend on it")
	}
}
