package http

import (
	"context"
	"customer-service/config"
	"customer-service/internal/adapters/http/handler"
	"customer-service/internal/domain/ports"
	"customer-service/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Server struct {
	httpServer *http.Server
	engine     *gin.Engine
}

func NewServer(cfg config.API, accountService ports.CustomerService) *Server {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	v1 := engine.Group("/api/v1")
	handler.NewCustomerHandler(accountService).RegisterRoutes(v1)

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
