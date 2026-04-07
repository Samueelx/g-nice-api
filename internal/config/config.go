package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	Port       string
	JWTSecret  string
	GinMode    string
	ResendAPIKey string
	EmailFrom    string
}

// Load reads environment variables and returns a Config.
// It returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "gnice_user"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "gnice_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		Port:       getEnv("PORT", "8080"),
		JWTSecret:    getEnv("JWT_SECRET", ""),
		GinMode:      getEnv("GIN_MODE", "debug"),
		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		EmailFrom:    getEnv("EMAIL_FROM", ""),
	}

	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.ResendAPIKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is required")
	}
	if cfg.EmailFrom == "" {
		return nil, fmt.Errorf("EMAIL_FROM is required")
	}

	return cfg, nil
}

// DSN returns the PostgreSQL data source name (connection string).
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
