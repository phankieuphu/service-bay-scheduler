package application

import (
	"context"
	"sync"
	"time"
	"vehicle-service/config"
	"vehicle-service/internal/adapters/cache"
	database_provider "vehicle-service/internal/adapters/database/provider"
	ginhttp "vehicle-service/internal/adapters/http"
	"vehicle-service/internal/adapters/http/handler"
	"vehicle-service/internal/adapters/kafka"
	"vehicle-service/internal/adapters/repository"
	"vehicle-service/internal/domain/services"
	"vehicle-service/pkg/logger"

	"github.com/joho/godotenv"
)

// outboxRelayInterval is how often the outbox table is polled for
// messages to publish to Kafka.
const outboxRelayInterval = 2 * time.Second

// shutdownTimeout bounds how long in-flight HTTP requests get to finish
// once a shutdown signal arrives, before the server is forcibly closed.
const shutdownTimeout = 15 * time.Second

type Application struct {
}

func VehicleApplication(ctx context.Context) {
	godotenv.Load()
	cfg := config.LoadConfig()
	logger.Init(cfg.Logger.ToOptions())

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

	// Redis cache
	redisCache, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		logger.Fatal("failed to init Redis", "error", err)
	}

	// repository & service
	txManager := database_provider.NewTxManager(database)
	entryVehicleRepository := repository.NewVehicleRepository(database)
	entryVehicleCustomerRepository := repository.NewVehicleCustomerRepository(database)
	entryOutboxRepository := repository.NewOutboxRepository(database)
	entryVehicleService := services.NewVehicleService(*cfg, entryVehicleRepository, entryOutboxRepository, entryVehicleCustomerRepository, txManager, redisCache)

	// Background workers run on their own context, cancelled only after the
	// HTTP server has drained, so the outbox relay keeps running while
	// in-flight requests are still writing outbox rows.
	workerCtx, stopWorkers := context.WithCancel(context.WithoutCancel(ctx))
	defer stopWorkers()
	var workers sync.WaitGroup

	// Outbox relay: delivers rows written by the service above to Kafka.
	outboxRelay := kafka.NewOutboxRelay(kafkaProducer, entryOutboxRepository, outboxRelayInterval)
	workers.Go(func() { outboxRelay.Start(workerCtx) })

	// Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka, func(ctx context.Context, key, value []byte) error {
		logger.Info("kafka message received", "key", string(key), "value", string(value))
		return nil
	})
	if err != nil {
		logger.Fatal("failed to init Kafka consumer", "error", err)
	}

	// HTTP server (gin)
	sqlDB, err := database.DB()
	if err != nil {
		logger.Fatal("failed to get database handle", "error", err)
	}
	health := handler.NewHealthHandler(
		handler.HealthCheck{Name: "postgres", Critical: true, Check: sqlDB.PingContext},
		handler.HealthCheck{Name: "kafka", Check: kafkaProducer.Ping},
		handler.HealthCheck{Name: "redis", Check: redisCache.Ping},
	)
	httpServer := ginhttp.NewServer(cfg.API, entryVehicleService, health)

	logger.Info("Vehicle Application Started")

	workers.Go(func() { kafkaConsumer.Start(workerCtx) })

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

	// 2. Stop the consumer and outbox relay, and wait for any in-progress
	//    message / relay batch to finish before closing what they use.
	stopWorkers()
	workers.Wait()

	// 3. Release connections.
	if err := kafkaConsumer.Close(); err != nil {
		logger.Error("close Kafka consumer", "error", err)
	}
	if err := kafkaProducer.Close(); err != nil {
		logger.Error("close Kafka producer", "error", err)
	}
	if err := redisCache.Close(); err != nil {
		logger.Error("close Redis", "error", err)
	}
	if err := sqlDB.Close(); err != nil {
		logger.Error("close database", "error", err)
	}

	logger.Info("Vehicle Application stopped")
}
