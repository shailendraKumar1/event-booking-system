package config

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// Config holds all application configuration. Singleton via Get().
type Config struct {
	Port            string
	JWTSecret       string
	JWTExpiry       time.Duration
	SMTPHost        string
	SMTPPort        string
	SMTPUsername    string
	SMTPPassword   string
	EmailFrom       string
	WorkerPoolSize  int
	JobQueueSize    int
	CacheTTL        time.Duration
	RateLimitRPS    int
	RateLimitBurst  int
}

var (
	instance *Config
	once     sync.Once
)

func Get() *Config {
	once.Do(func() {
		instance = &Config{
			Port:           env("PORT", "8080"),
			JWTSecret:      env("JWT_SECRET", "change-me-in-production-please"),
			JWTExpiry:      parseDuration("JWT_EXPIRY", 24*time.Hour),
			SMTPHost:       os.Getenv("SMTP_HOST"),
			SMTPPort:       env("SMTP_PORT", "587"),
			SMTPUsername:  os.Getenv("SMTP_USERNAME"),
			SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
			EmailFrom:      os.Getenv("EMAIL_FROM"),
			WorkerPoolSize: parseInt("WORKER_POOL_SIZE", 5),
			JobQueueSize:   parseInt("JOB_QUEUE_SIZE", 100),
			CacheTTL:       parseDuration("CACHE_TTL", 30*time.Second),
			RateLimitRPS:   parseInt("RATE_LIMIT_RPS", 100),
			RateLimitBurst: parseInt("RATE_LIMIT_BURST", 200),
		}
	})
	return instance
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
