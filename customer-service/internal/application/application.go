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
	"log"

	"github.com/joho/godotenv"
)

type Application struct {
}

func CustomerApplication(ctx context.Context) {
	godotenv.Load()
	cfg := config.LoadConfig()

	// database
	database, err := database_provider.NewPostgresClient(*cfg)
	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}

	// repository & service
	entryCustomerRepository := repository.NewCustomerRepository(database)
	entryCustomerService := services.NewCustomerService(*cfg, entryCustomerRepository)

	// Redis cache
	redisCache, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		log.Fatalf("failed to init Redis: %v", err)
	}
	_ = redisCache

	// Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		log.Fatalf("failed to init Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka, func(ctx context.Context, key, value []byte) error {
		log.Printf("kafka message received key=%s value=%s", key, value)
		return nil
	})
	if err != nil {
		log.Fatalf("failed to init Kafka consumer: %v", err)
	}
	defer kafkaConsumer.Close()

	// HTTP server (gin)
	httpServer := ginhttp.NewServer(cfg.API, entryCustomerService)

	log.Println("Customer Application Started")

	go kafkaConsumer.Start(ctx)
	httpServer.Start()
}
