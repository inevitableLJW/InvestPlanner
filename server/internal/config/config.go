package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment  string
	HTTPAddr     string
	MySQLDSN     string
	WebOrigin    string
	CookieSecure bool
	SessionTTL   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment: os.Getenv("APP_ENV"),
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
		MySQLDSN:    os.Getenv("MYSQL_DSN"),
		WebOrigin:   os.Getenv("WEB_ORIGIN"),
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.WebOrigin == "" {
		cfg.WebOrigin = "http://localhost:5173"
	}
	if cfg.MySQLDSN == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN is required")
	}
	if raw := os.Getenv("COOKIE_SECURE"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("COOKIE_SECURE: %w", err)
		}
		cfg.CookieSecure = value
	} else {
		cfg.CookieSecure = cfg.Environment == "production"
	}
	cfg.SessionTTL = 7 * 24 * time.Hour
	if raw := os.Getenv("SESSION_TTL"); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("SESSION_TTL: %w", err)
		}
		cfg.SessionTTL = value
	}
	return cfg, nil
}
