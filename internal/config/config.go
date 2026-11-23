package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config хранит все настройки приложения.
type Config struct {
	AppEnv      string
	IsProd      bool
	HTTPPort    string
	LogLevel    string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
}

// Load загружает конфигурацию из файла .env (если существует) и переменных окружения.
func Load() (*Config, error) {
	// Игнорируем ошибку отсутствия .env, так как в продакшене переменные передаются через окружение
	_ = godotenv.Overload()

	appEnv := getEnv("APP_ENV", "dev")
	isProd := strings.EqualFold(appEnv, "prod") || strings.EqualFold(appEnv, "production")

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}

	ttlStr := getEnv("JWT_TTL", "24h")
	jwtTTL, err := time.ParseDuration(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_TTL duration %q: %w", ttlStr, err)
	}

	return &Config{
		AppEnv:      appEnv,
		IsProd:      isProd,
		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
		JWTTTL:      jwtTTL,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return defaultValue
}
