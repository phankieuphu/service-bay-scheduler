package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Database
	API
	Kafka
	Redis
	Logger
}

type Database struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type API struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Kafka struct {
	Brokers []string
	// ProducerTopic is where this service publishes Customer lifecycle events
	// (CustomerCreated/CustomerUpdated) for other services to consume.
	ProducerTopic string
	// ConsumerTopic is identity-service's user-event stream, not our own
	// ProducerTopic — customer-service reacts to new/updated Users (role=Customer)
	// to create/update the local Customer profile. Must stay a different topic
	// from ProducerTopic or this consumer group would replay its own output.
	ConsumerTopic string
	ConsumerGroup string
}

type Redis struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func (r Redis) Addr() string {
	return r.Host + ":" + r.Port
}

type Logger struct {
	Level     string // debug, info, warn, error
	Format    string // json, text
	AddSource bool
}

func LoadConfig() *Config {
	return &Config{
		Database: Database{
			Host:            GetEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			Username:        GetEnv("DB_USERNAME", "postgres"),
			Password:        GetEnv("DB_PASSWORD", ""),
			Database:        GetEnv("DB_NAME", "customer_service"),
			SSLMode:         GetEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 1800)) * time.Second,
		},
		API: API{
			Port:         GetEnv("API_PORT", "8080"),
			ReadTimeout:  time.Duration(getEnvInt("API_READ_TIMEOUT_SEC", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("API_WRITE_TIMEOUT_SEC", 30)) * time.Second,
		},
		Kafka: Kafka{
			Brokers:       []string{GetEnv("KAFKA_BROKERS", "localhost:9092")},
			ProducerTopic: GetEnv("KAFKA_PRODUCER_TOPIC", "customer.events"),
			ConsumerTopic: GetEnv("KAFKA_CONSUMER_TOPIC", "identity.user-events"),
			ConsumerGroup: GetEnv("KAFKA_CONSUMER_GROUP", "customer-service"),
		},
		Redis: Redis{
			Host:     GetEnv("REDIS_HOST", "localhost"),
			Port:     GetEnv("REDIS_PORT", "6379"),
			Password: GetEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Logger: Logger{
			Level:     GetEnv("LOG_LEVEL", "info"),
			Format:    GetEnv("LOG_FORMAT", "json"),
			AddSource: getEnvBool("LOG_ADD_SOURCE", false),
		},
	}
}

func GetEnv(key string, defaultValue string) string {
	result := os.Getenv(key)
	if result != "" {
		return result
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return n
}

func getEnvBool(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return b
}
