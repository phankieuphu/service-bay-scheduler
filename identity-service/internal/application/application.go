package application

import (
	"context"
	"identity-service/config"
	"identity-service/internal/adapters/cache"
	database_provider "identity-service/internal/adapters/database/provider"
	ginhttp "identity-service/internal/adapters/http"
	"identity-service/internal/adapters/http/handler"
	"identity-service/internal/adapters/kafka"
	"identity-service/internal/adapters/repository"
	"identity-service/internal/adapters/security"
	"identity-service/internal/domain/services"
	"identity-service/pkg/logger"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// outboxRelayInterval is how often the outbox table is polled for
// messages to publish to Kafka.
const outboxRelayInterval = 2 * time.Second

// shutdownTimeout bounds how long in-flight HTTP requests get to finish
// once a shutdown signal arrives, before the server is forcibly closed.
const shutdownTimeout = 15 * time.Second

// bcryptCost is the password hashing work factor (2^12 rounds, ~250ms).
const bcryptCost = 12

func IdentityApplication(ctx context.Context) {
	godotenv.Load()
	cfg := config.LoadConfig()
	logger.Init(cfg.Logger.ToOptions())

	tokenIssuer, err := security.NewJWTIssuer(cfg.JWT)
	if err != nil {
		logger.Fatal("failed to init JWT issuer", "error", err)
	}

	// database
	database, err := database_provider.NewPostgresClient(*cfg)
	if err != nil {
		logger.Fatal("failed to init database", "error", err)
	}

	// Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		logger.Fatal("failed to init Kafka producer", "error", err)
	}

	// Redis refresh-token store
	redisCache, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		logger.Fatal("failed to init Redis", "error", err)
	}

	// repository & service
	txManager := database_provider.NewTxManager(database)
	entryUserRepository := repository.NewUserRepository(database)
	entryOutboxRepository := repository.NewOutboxRepository(database)
	entryAuthService, err := services.NewAuthService(*cfg, entryUserRepository, entryOutboxRepository, txManager,
		security.NewBcryptHasher(bcryptCost), tokenIssuer, redisCache)
	if err != nil {
		logger.Fatal("failed to init auth service", "error", err)
	}

	// Background workers run on their own context, cancelled only after the
	// HTTP server has drained, so the outbox relay keeps running while
	// in-flight requests are still writing outbox rows.
	workerCtx, stopWorkers := context.WithCancel(context.WithoutCancel(ctx))
	defer stopWorkers()
	var workers sync.WaitGroup

	// Outbox relay: delivers rows written by the service above to Kafka.
	outboxRelay := kafka.NewOutboxRelay(kafkaProducer, entryOutboxRepository, outboxRelayInterval)
	workers.Go(func() { outboxRelay.Start(workerCtx) })

	// HTTP server (gin)
	sqlDB, err := database.DB()
	if err != nil {
		logger.Fatal("failed to get database handle", "error", err)
	}
	health := handler.NewHealthHandler(
		handler.HealthCheck{Name: "postgres", Critical: true, Check: sqlDB.PingContext},
		handler.HealthCheck{Name: "kafka", Check: kafkaProducer.Ping},
		// Critical here, unlike in customer/vehicle-service: Redis holds the
		// refresh tokens, so login and refresh can't work without it.
		handler.HealthCheck{Name: "redis", Critical: true, Check: redisCache.Ping},
	)
	httpServer := ginhttp.NewServer(cfg.API, entryAuthService, tokenIssuer, health)

	logger.Info("Identity Application Started")

	serverErr := make(chan error, 1)
	go func() { serverErr <- httpServer.Start() }()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received, shutting down")
	case err := <-serverErr:
		if err != nil {
			logger.Error("HTTP server error, shutting down", "error", err)
		}
	}

	// 1. Stop accepting requests and let in-flight ones finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown", "error", err)
	}

	// 2. Stop the outbox relay and wait for any in-progress batch to finish
	//    before closing what it uses.
	stopWorkers()
	workers.Wait()

	// 3. Release connections.
	if err := kafkaProducer.Close(); err != nil {
		logger.Error("close Kafka producer", "error", err)
	}
	if err := redisCache.Close(); err != nil {
		logger.Error("close Redis", "error", err)
	}
	if err := sqlDB.Close(); err != nil {
		logger.Error("close database", "error", err)
	}

	logger.Info("Identity Application stopped")
}
