package http

import (
	"context"
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

func NewServer(cfg config.API, accountService ports.VehicleService) *Server {
	engine := gin.New()
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

func (s *Server) Start() {
	logger.Info("HTTP server listening", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("HTTP server error", "error", err)
	}
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
