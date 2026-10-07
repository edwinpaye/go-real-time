package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration properties for the enterprise sales platform.
type Config struct {
	AppEnv         string
	Port           string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	DBMaxOpenConns int
	DBMaxIdleConns int
	DBConnMaxLife  time.Duration
	JWTSecret      string
	JWTExpiryHours int
	StaticPath     string
	StaticPrefix   string
	WorkerPoolSize int
	WorkerQueueCap int
}

// LoadConfig reads configuration values from environment variables or applies production-safe defaults.
func LoadConfig() *Config {
	return &Config{
		AppEnv:         getEnv("APP_ENV", "production"),
		Port:           getEnv("PORT", "8080"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBName:         getEnv("DB_NAME", "sales_db"),
		DBSSLMode:      getEnv("DB_SSL_MODE", "disable"),
		DBMaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLife:  time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINS", 15)) * time.Minute,
		JWTSecret:      getEnv("JWT_SECRET", "super-secret-enterprise-sales-jwt-key-32bytes"),
		JWTExpiryHours: getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		StaticPath:     getEnv("STATIC_PATH", "./web"),
		StaticPrefix:   getEnv("STATIC_PREFIX", "/app/"),
		WorkerPoolSize: getEnvAsInt("WORKER_POOL_SIZE", 8),
		WorkerQueueCap: getEnvAsInt("WORKER_QUEUE_CAP", 1000),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	strValue := getEnv(key, "")
	if strValue == "" {
		return fallback
	}
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return fallback
}
