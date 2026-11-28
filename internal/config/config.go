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
	AppEnv          string
	IsProd          bool
	HTTPPort        string
	LogLevel        string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	AdminEmail      string
	UploadDir       string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPassword    string
	SMTPFrom        string
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

	accessTTLStr := getEnv("ACCESS_TOKEN_TTL", "15m")
	accessTokenTTL, err := time.ParseDuration(accessTTLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ACCESS_TOKEN_TTL duration %q: %w", accessTTLStr, err)
	}

	refreshTTLStr := getEnv("REFRESH_TOKEN_TTL", "720h")
	refreshTokenTTL, err := time.ParseDuration(refreshTTLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REFRESH_TOKEN_TTL duration %q: %w", refreshTTLStr, err)
	}

	smtpHost := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	smtpPort := getEnv("SMTP_PORT", "1025")
	smtpUser := strings.TrimSpace(os.Getenv("SMTP_USER"))
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpFrom := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if isProd {
		if smtpHost == "" {
			return nil, errors.New("SMTP_HOST is required in production")
		}
		if smtpFrom == "" {
			return nil, errors.New("SMTP_FROM is required in production")
		}
	} else if smtpFrom == "" {
		smtpFrom = "no-reply@bboard.local"
	}

	return &Config{
		AppEnv:          appEnv,
		IsProd:          isProd,
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DatabaseURL:     dbURL,
		JWTSecret:       jwtSecret,
		AccessTokenTTL:  accessTokenTTL,
		RefreshTokenTTL: refreshTokenTTL,
		AdminEmail:      strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		UploadDir:       getEnv("UPLOAD_DIR", "./uploads"),
		SMTPHost:        smtpHost,
		SMTPPort:        smtpPort,
		SMTPUser:        smtpUser,
		SMTPPassword:    smtpPassword,
		SMTPFrom:        smtpFrom,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return defaultValue
}
