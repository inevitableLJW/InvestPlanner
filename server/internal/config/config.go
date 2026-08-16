package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
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
		var err error
		cfg.MySQLDSN, err = mysqlDSNFromEnvironment()
		if err != nil {
			return Config{}, err
		}
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

func mysqlDSNFromEnvironment() (string, error) {
	password := os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("MYSQL_DSN or MYSQL_PASSWORD is required")
	}

	host := envOrDefault("MYSQL_HOST", "mysql")
	port := envOrDefault("MYSQL_PORT", "3306")
	database := envOrDefault("MYSQL_DATABASE", "invest_planner")
	user := envOrDefault("MYSQL_USER", "invest_planner")
	dsn := mysql.NewConfig()
	dsn.User = user
	dsn.Passwd = password
	dsn.Net = "tcp"
	dsn.Addr = net.JoinHostPort(host, port)
	dsn.DBName = database
	dsn.Params = map[string]string{"charset": "utf8mb4"}
	dsn.ParseTime = true
	dsn.Loc = time.Local
	return dsn.FormatDSN(), nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
