package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	ServerPort         string
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	ServerIdleTimeout  time.Duration

	LogLevel    string
	Environment string

	MigrationsPath string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		DBHost:     getEnv("DB_HOST", "db"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "pr_review_service"),

		ServerPort: getEnv("SERVER_PORT", "8080"),

		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Environment: getEnv("ENVIRONMENT", "development"),

		MigrationsPath: getEnv("MIGRATIONS_PATH", "file://migrations"),
	}

	readTimeout, err := strconv.Atoi(getEnv("SERVER_READ_TIMEOUT", "15"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_READ_TIMEOUT: %w", err)
	}
	config.ServerReadTimeout = time.Duration(readTimeout) * time.Second

	writeTimeout, err := strconv.Atoi(getEnv("SERVER_WRITE_TIMEOUT", "15"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_WRITE_TIMEOUT: %w", err)
	}
	config.ServerWriteTimeout = time.Duration(writeTimeout) * time.Second

	idleTimeout, err := strconv.Atoi(getEnv("SERVER_IDLE_TIMEOUT", "60"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_IDLE_TIMEOUT: %w", err)
	}
	config.ServerIdleTimeout = time.Duration(idleTimeout) * time.Second

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) GetDBConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func (c *Config) GetMigrationConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}
