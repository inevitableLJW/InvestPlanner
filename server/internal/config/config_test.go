package config

import (
	"testing"

	"github.com/go-sql-driver/mysql"
)

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
