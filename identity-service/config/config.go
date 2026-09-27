package config

import (
	"identity-service/internal/constants"
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
	JWT
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
	// ProducerTopic is the user-event stream customer-service and
	// vehicle-service consume (their KAFKA_CONSUMER_TOPIC). identity-service
	// is upstream of everyone, so it has no consumer topic of its own.
	ProducerTopic string
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

type JWT struct {
	// Secret signs access tokens (HS256). Every service that validates
	// tokens itself needs the same value, so it comes from the environment
	// and has no default.
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func LoadConfig() *Config {
	return &Config{
		Database: Database{
			Host:            GetEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			Username:        GetEnv("DB_USERNAME", "postgres"),
			Password:        GetEnv("DB_PASSWORD", ""),
			Database:        GetEnv("DB_NAME", "identity_db"),
			SSLMode:         GetEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 1800)) * time.Second,
		},
		API: API{
			Port:         GetEnv("API_PORT", "8083"),
			ReadTimeout:  time.Duration(getEnvInt("API_READ_TIMEOUT_SEC", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("API_WRITE_TIMEOUT_SEC", 30)) * time.Second,
		},
		Kafka: Kafka{
			Brokers:       []string{GetEnv("KAFKA_BROKERS", "localhost:9092")},
			ProducerTopic: GetEnv("KAFKA_PRODUCER_TOPIC", constants.UserEventsTopic),
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
		JWT: JWT{
			Secret:     GetEnv("JWT_SECRET", ""),
			Issuer:     GetEnv("JWT_ISSUER", "identity-service"),
			AccessTTL:  time.Duration(getEnvInt("JWT_ACCESS_TTL_SEC", 900)) * time.Second,
			RefreshTTL: time.Duration(getEnvInt("JWT_REFRESH_TTL_SEC", 604800)) * time.Second,
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
