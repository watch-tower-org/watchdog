package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	JWT          JWTConfig
	Admin        AdminConfig
	RateLimit    RateLimitConfig
	CORS         CORSConfig
	LoggerConfig LoggerConfig
}

type ServerConfig struct {
	Port     string
	Mode     string
	TimeZone string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	SecretKey                 string
	ExpirationDuration        time.Duration
	RefreshSecretKey          string
	RefreshExpirationDuration time.Duration
}

type AdminConfig struct {
	Username string
	Password string
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

type LoggerConfig struct {
	Output       OutputMode
	Level        string
	Filename     string
	LogDir       string
	MaxSizeMB    int
	MaxBackups   int
	MaxAgeDays   int
	Compress     bool
	EnableCaller bool
}

type OutputMode string

const (
	OutputStdout OutputMode = "stdout"
	OutputFile   OutputMode = "file"
	OutputBoth   OutputMode = "both"
)

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Print("Error loading .env file\n")
		fmt.Print("Attempting to read configuration from system environment variables\n")
	}

	config := &Config{
		Server: ServerConfig{
			Port:     getEnv("SERVER_PORT", "8080"),
			Mode:     getEnv("GIN_MODE", "release"),
			TimeZone: getEnv("TZ", "UTC"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "watchtower"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		JWT: JWTConfig{
			SecretKey:                 getEnv("JWT_SECRET_KEY", "watchtower-secret-key"),
			ExpirationDuration:        getEnvDuration("JWT_EXPIRATION_DURATION", 24*time.Hour),
			RefreshSecretKey:          getEnv("JWT_REFRESH_SECRET_KEY", "watchtower-refresh-secret-key"),
			RefreshExpirationDuration: getEnvDuration("JWT_REFRESH_EXPIRATION_DURATION", 168*time.Hour),
		},
		Admin: AdminConfig{
			Username: getEnv("ADMIN_USERNAME", "admin"),
			Password: getEnv("ADMIN_PASSWORD", "admin"),
		},
		RateLimit: RateLimitConfig{
			Requests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
			Window:   getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
			AllowedMethods: getEnvSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
			AllowedHeaders: getEnvSlice("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"}),
		},
		LoggerConfig: LoggerConfig{
			Output:       OutputMode(getEnv("LOGGER_OUTPUT", "stdout")),
			Level:        getEnv("LOGGER_LEVEL", "info"),
			Filename:     getEnv("LOGGER_FILENAME", "watchtower"),
			LogDir:       getEnv("LOGGER_LOG_DIR", "logs"),
			MaxSizeMB:    getEnvInt("LOGGER_MAX_SIZE_MB", 100),
			MaxBackups:   getEnvInt("LOGGER_MAX_BACKUPS", 7),
			MaxAgeDays:   getEnvInt("LOGGER_MAX_AGE_DAYS", 30),
			Compress:     getEnv("LOGGER_COMPRESS", "false") == "true",
			EnableCaller: getEnv("LOGGER_ENABLE_CALLER", "false") == "true",
		},
	}
	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	fmt.Printf("Using default value for key: %v -> %v\n", key, defaultValue)
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	fmt.Printf("Using default value for key: %v -> %v\n", key, defaultValue)
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	fmt.Printf("Using default value for key: %v -> %v\n", key, defaultValue)
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Printf("Using default value for key: %v -> %v\n", key, defaultValue)
		return defaultValue
	}

	values := strings.Split(value, ",")

	for i, v := range values {
		values[i] = strings.TrimSpace(v)
	}

	return values
}
