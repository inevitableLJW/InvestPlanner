package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type CookieSecureMode uint8

const (
	CookieSecureAuto CookieSecureMode = iota
	CookieSecureAlways
	CookieSecureNever
)

type Config struct {
	Environment                  string
	HTTPAddr                     string
	MySQLDSN                     string
	WebOrigin                    string
	CookieSecureMode             CookieSecureMode
	SessionTTL                   time.Duration
	AKToolsBaseURL               string
	TushareToken                 string
	ChinaMoneyURL                string
	MarketDataProductionApproved bool
	FXAssistantEnabled           bool
}

func Load() (Config, error) {
	cfg := Config{
		Environment:    os.Getenv("APP_ENV"),
		HTTPAddr:       os.Getenv("HTTP_ADDR"),
		MySQLDSN:       os.Getenv("MYSQL_DSN"),
		WebOrigin:      os.Getenv("WEB_ORIGIN"),
		AKToolsBaseURL: envOrDefault("AKTOOLS_BASE_URL", "http://aktools:8080"),
		TushareToken:   os.Getenv("TUSHARE_TOKEN"),
		ChinaMoneyURL:  os.Getenv("CHINAMONEY_URL"),
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
		value, err := parseCookieSecureMode(raw)
		if err != nil {
			return Config{}, fmt.Errorf("COOKIE_SECURE: %w", err)
		}
		cfg.CookieSecureMode = value
	}
	cfg.SessionTTL = time.Hour
	if raw := os.Getenv("SESSION_TTL"); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("SESSION_TTL: %w", err)
		}
		cfg.SessionTTL = value
	}
	var err error
	if cfg.MarketDataProductionApproved, err = envBool("MARKET_DATA_PRODUCTION_APPROVED", false); err != nil {
		return Config{}, err
	}
	if cfg.FXAssistantEnabled, err = envBool("FX_ASSISTANT_ENABLED", true); err != nil {
		return Config{}, err
	}
	if cfg.Environment == "production" && !cfg.MarketDataProductionApproved {
		return Config{}, fmt.Errorf("MARKET_DATA_PRODUCTION_APPROVED must be true because QDII market data is always enabled")
	}
	return cfg, nil
}

func envBool(name string, fallback bool) (bool, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: must be true or false", name)
	}
	return value, nil
}

func parseCookieSecureMode(raw string) (CookieSecureMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "auto":
		return CookieSecureAuto, nil
	case "true":
		return CookieSecureAlways, nil
	case "false":
		return CookieSecureNever, nil
	default:
		return CookieSecureAuto, fmt.Errorf("must be auto, true, or false")
	}
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
