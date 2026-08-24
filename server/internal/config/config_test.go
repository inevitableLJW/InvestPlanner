package config

import (
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestParseCookieSecureMode(t *testing.T) {
	tests := []struct {
		raw  string
		want CookieSecureMode
	}{
		{raw: "auto", want: CookieSecureAuto},
		{raw: "true", want: CookieSecureAlways},
		{raw: "false", want: CookieSecureNever},
	}
	for _, tt := range tests {
		got, err := parseCookieSecureMode(tt.raw)
		if err != nil || got != tt.want {
			t.Fatalf("parseCookieSecureMode(%q) = %v, %v; want %v", tt.raw, got, err, tt.want)
		}
	}
	if _, err := parseCookieSecureMode("HTTPS"); err == nil {
		t.Fatal("expected invalid COOKIE_SECURE mode to fail")
	}
}

func TestLoadDefaultsToOneHourAndAutomaticSecureCookie(t *testing.T) {
	t.Setenv("MYSQL_DSN", "user:password@tcp(127.0.0.1:3306)/planner?parseTime=true")
	t.Setenv("SESSION_TTL", "")
	t.Setenv("COOKIE_SECURE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SessionTTL != time.Hour {
		t.Fatalf("SessionTTL = %s, want 1h", cfg.SessionTTL)
	}
	if cfg.CookieSecureMode != CookieSecureAuto {
		t.Fatalf("CookieSecureMode = %v, want auto", cfg.CookieSecureMode)
	}
}

func TestMySQLDSNFromEnvironment(t *testing.T) {
	t.Setenv("MYSQL_HOST", "db.internal")
	t.Setenv("MYSQL_PORT", "3307")
	t.Setenv("MYSQL_DATABASE", "planner")
	t.Setenv("MYSQL_USER", "planner_user")
	t.Setenv("MYSQL_PASSWORD", "p@ss:word/with?symbols")

	raw, err := mysqlDSNFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := mysql.ParseDSN(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Addr != "db.internal:3307" || parsed.DBName != "planner" || parsed.User != "planner_user" || parsed.Passwd != "p@ss:word/with?symbols" {
		t.Fatalf("unexpected DSN fields: %#v", parsed)
	}
	if !parsed.ParseTime || parsed.Params["charset"] != "utf8mb4" {
		t.Fatalf("expected parseTime and utf8mb4: %#v", parsed)
	}
}

func TestMySQLDSNFromEnvironmentRequiresPassword(t *testing.T) {
	t.Setenv("MYSQL_PASSWORD", "")
	if _, err := mysqlDSNFromEnvironment(); err == nil {
		t.Fatal("expected missing password error")
	}
}

func TestProductionMarketDataRequiresExplicitApproval(t *testing.T) {
	t.Setenv("MYSQL_DSN", "user:password@tcp(127.0.0.1:3306)/planner?parseTime=true")
	t.Setenv("APP_ENV", "production")
	t.Setenv("MARKET_DATA_PRODUCTION_APPROVED", "false")
	if _, err := Load(); err == nil {
		t.Fatal("expected always-on QDII production authorization gate")
	}
	t.Setenv("MARKET_DATA_PRODUCTION_APPROVED", "true")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}

func TestQDIIDataHasNoDisableFlag(t *testing.T) {
	t.Setenv("MYSQL_DSN", "user:password@tcp(127.0.0.1:3306)/planner?parseTime=true")
	t.Setenv("APP_ENV", "development")
	t.Setenv("QDII_ASSISTANT_ENABLED", "false")
	t.Setenv("MARKET_DATA_ENABLED", "false")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}
