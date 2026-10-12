package config

import (
	"identity-service/internal/constants"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Database
	API
	Kafka
	Redis
	Logger
	JWT
	RateLimit
	Admin
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
	// TrustedProxies are the proxy IPs/CIDRs whose X-Forwarded-For is
	// believed when working out the client IP for rate limiting. Empty
	// means none: the TCP peer address is the client, so a client can't
	// dodge the per-IP limits by sending its own X-Forwarded-For.
	TrustedProxies []string
	// AllowedOrigins are the browser origins allowed to call this API with
	// credentials (the refresh-token cookie). A cookie rules out the "*"
	// wildcard, so they must be listed.
	AllowedOrigins []string
	// CookieSecure sets Secure on the refresh-token cookie. Only turn it
	// off for local plain-HTTP setups that won't send Secure cookies (e.g.
	// Safari or Postman against http://localhost).
	CookieSecure bool
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

// RateLimit caps login and registration attempts (fixed windows, counted
// in Redis). Login counts failures only, per IP and per email; a
// successful login resets that email's count. Register counts every
// attempt per IP.
type RateLimit struct {
	LoginPerEmail  int
	LoginPerIP     int
	LoginWindow    time.Duration
	RegisterPerIP  int
	RegisterWindow time.Duration
}

// Admin bootstraps the first ADMIN account on startup when both fields are
// set and no admin exists yet (see AdminService.BootstrapAdmin). Unset
// them once it exists; they're ignored from then on.
type Admin struct {
	Email    string
	Password string
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
			AutoMigrate:     getEnvBool("DB_AUTO_MIGRATE", true),
		},
		API: API{
			Port:           GetEnv("API_PORT", "8083"),
			ReadTimeout:    time.Duration(getEnvInt("API_READ_TIMEOUT_SEC", 30)) * time.Second,
			WriteTimeout:   time.Duration(getEnvInt("API_WRITE_TIMEOUT_SEC", 30)) * time.Second,
			TrustedProxies: getEnvList("API_TRUSTED_PROXIES"),
			AllowedOrigins: getEnvListDefault("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
			CookieSecure:   getEnvBool("AUTH_COOKIE_SECURE", true),
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
		RateLimit: RateLimit{
			LoginPerEmail:  getEnvInt("RATE_LIMIT_LOGIN_PER_EMAIL", 10),
			LoginPerIP:     getEnvInt("RATE_LIMIT_LOGIN_PER_IP", 100),
			LoginWindow:    time.Duration(getEnvInt("RATE_LIMIT_LOGIN_WINDOW_SEC", 900)) * time.Second,
			RegisterPerIP:  getEnvInt("RATE_LIMIT_REGISTER_PER_IP", 20),
			RegisterWindow: time.Duration(getEnvInt("RATE_LIMIT_REGISTER_WINDOW_SEC", 3600)) * time.Second,
		},
		Admin: Admin{
			Email:    GetEnv("ADMIN_EMAIL", ""),
			Password: GetEnv("ADMIN_PASSWORD", ""),
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

// getEnvListDefault is getEnvList, or defaultValue when the variable is
// unset or empty.
func getEnvListDefault(key string, defaultValue []string) []string {
	if list := getEnvList(key); len(list) > 0 {
		return list
	}
	return defaultValue
}

// getEnvList splits a comma-separated variable, dropping empty entries.
func getEnvList(key string) []string {
	var out []string
	for _, v := range strings.Split(os.Getenv(key), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
