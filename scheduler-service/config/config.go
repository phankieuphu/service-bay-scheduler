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
	Upstream
	Logger
	JWT
	Catalog
}

// Catalog describes the service catalog's prices. Currency is the ISO 4217
// code every fee is in; it travels with ServiceCompleted and the appointment
// events so billing-service never has to guess it.
type Catalog struct {
	Currency string
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
	// AutoMigrate applies pending migrations (migrations/*.sql) on startup.
	// Turn it off to run them separately, e.g. with cmd/migrate as a k8s
	// Job before a rollout.
	AutoMigrate bool
}

type API struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Kafka is producer-only: scheduler-service publishes its appointment
// events (constants/topics.go) through the outbox and consumes nothing yet.
type Kafka struct {
	Brokers []string
}

// Upstream is where scheduler-service checks, at booking time, who the
// caller is as a customer and which vehicles they own (architecture-design.md
// §4 option 1, §5a). Both calls forward the caller's own access token.
type Upstream struct {
	CustomerAPIURL string
	VehicleAPIURL  string
	Timeout        time.Duration
}

type Logger struct {
	Level     string // debug, info, warn, error
	Format    string // json, text
	AddSource bool
}

// JWT verifies identity-service's access tokens. Secret and Issuer must
// match identity-service's JWT_SECRET and JWT_ISSUER.
type JWT struct {
	Secret string
	Issuer string
}

func LoadConfig() *Config {
	return &Config{
		Database: Database{
			Host:            GetEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			Username:        GetEnv("DB_USERNAME", "postgres"),
			Password:        GetEnv("DB_PASSWORD", ""),
			Database:        GetEnv("DB_NAME", "scheduler_db"),
			SSLMode:         GetEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 1800)) * time.Second,
			AutoMigrate:     getEnvBool("DB_AUTO_MIGRATE", true),
		},
		API: API{
			Port:         GetEnv("API_PORT", "8085"),
			ReadTimeout:  time.Duration(getEnvInt("API_READ_TIMEOUT_SEC", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("API_WRITE_TIMEOUT_SEC", 30)) * time.Second,
		},
		Kafka: Kafka{
			Brokers: []string{GetEnv("KAFKA_BROKERS", "localhost:9092")},
		},
		Upstream: Upstream{
			CustomerAPIURL: GetEnv("CUSTOMER_API_URL", "http://localhost:8080/api/v1"),
			VehicleAPIURL:  GetEnv("VEHICLE_API_URL", "http://localhost:8081/api/v1"),
			Timeout:        time.Duration(getEnvInt("UPSTREAM_TIMEOUT_SEC", 5)) * time.Second,
		},
		Logger: Logger{
			Level:     GetEnv("LOG_LEVEL", "info"),
			Format:    GetEnv("LOG_FORMAT", "json"),
			AddSource: getEnvBool("LOG_ADD_SOURCE", false),
		},
		JWT: JWT{
			Secret: GetEnv("JWT_SECRET", ""),
			Issuer: GetEnv("JWT_ISSUER", "identity-service"),
		},
		Catalog: Catalog{
			Currency: GetEnv("CATALOG_CURRENCY", "USD"),
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
