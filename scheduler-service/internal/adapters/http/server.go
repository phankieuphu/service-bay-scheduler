package http

import (
	"context"
	"errors"
	"net/http"
	"scheduler-service/config"
	"scheduler-service/internal/adapters/http/handler"
	"scheduler-service/internal/adapters/http/middleware"
	"scheduler-service/internal/adapters/metrics"
	"scheduler-service/internal/domain/ports"
	"scheduler-service/pkg/logger"

	"github.com/gin-gonic/gin"
	health "github.com/phankieuphu/go-health-check"
	"github.com/phankieuphu/go-health-check/ginhealth"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	httpServer *http.Server
	engine     *gin.Engine
}

func NewServer(cfg config.API, bookingService ports.BookingService, tokens middleware.TokenVerifier, h *health.Handler) *Server {
	engine := gin.New()
	// Probes are registered before engine.Use, so they skip the middleware:
	// kubelet hits them every few seconds, which would flood the access log
	// and the request metrics.
	ginhealth.RegisterRoutes(engine, h)
	engine.Use(gin.Logger(), gin.Recovery())
	engine.Use(corsMiddleware())
	engine.Use(metrics.PrometheusMiddleWare())
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	// Every API route needs an identity-service access token; the probes
	// and /metrics above stay open.
	v1 := engine.Group("/api/v1", middleware.RequireAuth(tokens))
	handler.NewBookingHandler(bookingService).RegisterRoutes(v1)

	return &Server{
		engine: engine,
		httpServer: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      engine,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
	}
}

// Start blocks serving requests until Shutdown is called (returning nil)
// or the listener fails (returning the error).
func (s *Server) Start() error {
	logger.Info("HTTP server listening", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// corsMiddleware allows the frontend dev server (a different origin) to call
// this API directly; there's no gateway in front of these services yet.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
