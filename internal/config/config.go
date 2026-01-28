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
	Env                 string
	HTTPAddr            string
	DBURL               string
	JWTSecret           string
	JWTExpiry           time.Duration
	AllowedOrigins      []string
	RateLimitPerMinute  int
	RequestTimeout      time.Duration
	PasswordMinLen      int
	EnableDevResetCodes bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	env := getEnv("ENV", "dev")
	jwtExpiry := getDurationEnv("JWT_EXPIRES_IN", time.Hour)
	rateLimit := getIntEnv("RATE_LIMIT_PER_MIN", 30)
	requestTimeout := getDurationEnv("REQUEST_TIMEOUT", 5*time.Second)

	passwordMin := 4
	if env == "prod" {
		passwordMin = 8
	}

	allowedOrigins := splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"))
	cfg := &Config{
		Env:                 env,
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		DBURL:               getEnv("DATABASE_URL", "postgres://app:app@localhost:5432/avagostar?sslmode=disable"),
		JWTSecret:           getEnv("JWT_SECRET", "change-me"),
		JWTExpiry:           jwtExpiry,
		AllowedOrigins:      allowedOrigins,
		RateLimitPerMinute:  rateLimit,
		RequestTimeout:      requestTimeout,
		PasswordMinLen:      passwordMin,
		EnableDevResetCodes: env != "prod",
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var out []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
