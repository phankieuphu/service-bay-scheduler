package application

import (
	"context"
	"customer-service/config"
	"customer-service/internal/adapters/cache"
	database_provider "customer-service/internal/adapters/database/provider"
	ginhttp "customer-service/internal/adapters/http"
	"customer-service/internal/adapters/kafka"
	"customer-service/internal/adapters/repository"
	"customer-service/internal/domain/services"
	"customer-service/pkg/logger"
	"time"

	"github.com/joho/godotenv"
)

// outboxRelayInterval is how often the outbox table is polled for
// messages to publish to Kafka.
const outboxRelayInterval = 2 * time.Second

type Application struct {
}

func CustomerApplication(ctx context.Context) {
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
	defer kafkaProducer.Close()

	// Redis cache
	redisCache, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		logger.Fatal("failed to init Redis", "error", err)
	}
	defer redisCache.Close()

	// repository & service
	txManager := database_provider.NewTxManager(database)
	entryCustomerRepository := repository.NewCustomerRepository(database)
	entryOutboxRepository := repository.NewOutboxRepository(database)
	entryCustomerService := services.NewCustomerService(*cfg, entryCustomerRepository, entryOutboxRepository, txManager, redisCache)

	// Outbox relay: delivers rows written by the service above to Kafka.
	outboxRelay := kafka.NewOutboxRelay(kafkaProducer, entryOutboxRepository, outboxRelayInterval)
	go outboxRelay.Start(ctx)

	// Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka, func(ctx context.Context, key, value []byte) error {
		logger.Info("kafka message received", "key", string(key), "value", string(value))
		return nil
	})
	if err != nil {
		logger.Fatal("failed to init Kafka consumer", "error", err)
	}
	defer kafkaConsumer.Close()

	// HTTP server (gin)
	httpServer := ginhttp.NewServer(cfg.API, entryCustomerService)

	logger.Info("Customer Application Started")

	go kafkaConsumer.Start(ctx)
	httpServer.Start()
}
