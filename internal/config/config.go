package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI       string
	MongoDatabase  string
	MongoTimeout   time.Duration
	ServerAddr     string
	AuthToken      string
	CacheTTL       time.Duration
	MaxDateRange   time.Duration
	UseRedisCache  bool
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
}

func Load() (Config, error) {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	mongoTimeout, err := parseDurationEnv("MONGO_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	cacheTTL, err := parseDurationEnv("CACHE_TTL", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	maxRangeDays := getIntEnv("MAX_DATE_RANGE_DAYS", 366)

	cfg := Config{
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase: getEnv("MONGO_DATABASE", "jackpot"),
		MongoTimeout:  mongoTimeout,
		ServerAddr:    getEnv("SERVER_ADDR", ":8080"),
		AuthToken:     getEnv("ADMIN_AUTH_TOKEN", "Bearer secret-admin-token"),
		CacheTTL:      cacheTTL,
		MaxDateRange:  time.Duration(maxRangeDays) * 24 * time.Hour,
		UseRedisCache: getBoolEnv("USE_REDIS_CACHE", false),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getIntEnv("REDIS_DB", 0),
	}

	if cfg.AuthToken == "" {
		return Config{}, fmt.Errorf("ADMIN_AUTH_TOKEN cannot be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func parseDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration in %s: %w", key, err)
	}

	return d, nil
}

