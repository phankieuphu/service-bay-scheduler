package http

import (
	"context"
	"errors"
	"net/http"
	"vehicle-service/config"
	"vehicle-service/internal/adapters/http/handler"
	"vehicle-service/internal/adapters/metrics"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	httpServer *http.Server
	engine     *gin.Engine
}

func NewServer(cfg config.API, accountService ports.VehicleService, health *handler.HealthHandler) *Server {
	engine := gin.New()
	// Probes are registered before engine.Use, so they skip the middleware:
	// kubelet hits them every few seconds, which would flood the access log
	// and the request metrics.
	health.RegisterRoutes(engine)
	engine.Use(gin.Logger(), gin.Recovery())
	engine.Use(corsMiddleware())
	engine.Use(metrics.PrometheusMiddleWare())
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	v1 := engine.Group("/api/v1")
	handler.NewVehicleHandler(accountService).RegisterRoutes(v1)

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
