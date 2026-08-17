package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mysqlconfig "github.com/go-sql-driver/mysql"

	"investplanner/server/internal/config"
	"investplanner/server/internal/database"
	"investplanner/server/internal/httpapi"
	"investplanner/server/internal/repository"
	"investplanner/server/internal/service"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not configured")
	}
	if err := validateTestDSN(dsn); err != nil {
		t.Fatal(err)
	}
	return dsn
}

func validateTestDSN(dsn string) error {
	parsed, err := mysqlconfig.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("parse MYSQL_TEST_DSN: %w", err)
	}
	if !strings.HasSuffix(strings.ToLower(parsed.DBName), "_test") {
		return fmt.Errorf("refusing destructive integration tests on database %q: MYSQL_TEST_DSN must select a dedicated database ending in _test", parsed.DBName)
	}
	return nil
}

func TestValidateTestDSNRejectsDevelopmentDatabase(t *testing.T) {
	development := "user:password@tcp(127.0.0.1:3306)/invest_planner?parseTime=True"
	if err := validateTestDSN(development); err == nil {
		t.Fatal("expected development database DSN to be rejected")
	}
	testDatabase := "user:password@tcp(127.0.0.1:3306)/invest_planner_test?parseTime=True"
	if err := validateTestDSN(testDatabase); err != nil {
		t.Fatalf("expected dedicated test database DSN to be accepted: %v", err)
	}
}

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	dsn := testDSN(t)
	db, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS=0").Error; err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{
		"monthly_destination_allocations", "monthly_expenses", "monthly_records",
		"plan_destinations", "plans", "expense_sources", "sessions", "users",
	} {
		if err := db.Exec("TRUNCATE TABLE " + table).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS=1").Error; err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Environment: "test", WebOrigin: "http://localhost:5173", SessionTTL: time.Hour,
	}
	return httpapi.New(cfg, service.New(repository.New(db), time.Hour))
}

func requestJSON(t *testing.T, router http.Handler, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, &payload)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:5173")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func register(t *testing.T, router http.Handler, username string) *http.Cookie {
	t.Helper()
	response := requestJSON(t, router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username": username, "password": "very-secure-password1",
	}, nil)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].HttpOnly {
		t.Fatalf("expected HttpOnly session cookie: %#v", cookies)
	}
	return cookies[0]
}

func TestAuthenticationLifecycleAndIsolation(t *testing.T) {
	router := testRouter(t)
	first := register(t, router, "first-user")
	duplicate := requestJSON(t, router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username": "FIRST-USER", "password": "very-secure-password1",
	}, nil)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}

	created := requestJSON(t, router, http.MethodPost, "/api/v1/plans", map[string]any{"name": "长期计划"}, first)
	if created.Code != http.StatusCreated {
		t.Fatalf("create plan status=%d body=%s", created.Code, created.Body.String())
	}
	var plan map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	planID := plan["id"].(string)

	second := register(t, router, "second-user")
	crossUser := requestJSON(t, router, http.MethodGet, "/api/v1/plans/"+planID, nil, second)
	if crossUser.Code != http.StatusNotFound {
		t.Fatalf("cross-user status=%d body=%s", crossUser.Code, crossUser.Body.String())
	}

	logout := requestJSON(t, router, http.MethodPost, "/api/v1/auth/logout", nil, first)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body.String())
	}
	afterLogout := requestJSON(t, router, http.MethodGet, "/api/v1/plans", nil, first)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("after logout status=%d", afterLogout.Code)
	}
}

func TestWeakPasswordAndGenericLoginFailure(t *testing.T) {
	router := testRouter(t)
	weak := requestJSON(t, router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username": "weak-user", "password": "short",
	}, nil)
	if weak.Code != http.StatusBadRequest {
		t.Fatalf("weak status=%d body=%s", weak.Code, weak.Body.String())
	}
	login := requestJSON(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "missing-user", "password": "very-secure-password1",
	}, nil)
	if login.Code != http.StatusUnauthorized || !bytes.Contains(login.Body.Bytes(), []byte("用户名或密码错误")) {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
}

func TestHealth(t *testing.T) {
	router := testRouter(t)
	response := requestJSON(t, router, http.MethodGet, "/healthz", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatal(fmt.Sprintf("health status=%d", response.Code))
	}
}
func TestSessionRestoreCookieAndExpiry(t *testing.T) {
	router := testRouter(t)
	first := register(t, router, "session-user")
	if first.SameSite != http.SameSiteLaxMode || first.Secure {
		t.Fatalf("unexpected development cookie attributes: %#v", first)
	}
	restored := requestJSON(t, router, http.MethodGet, "/api/v1/auth/me", nil, first)
	if restored.Code != http.StatusOK {
		t.Fatalf("session restore status=%d body=%s", restored.Code, restored.Body.String())
	}
	expiring := register(t, router, "expiring-user")
	db, err := database.Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&database.Session{}).
		Where("token_hash = ?", service.HashToken(expiring.Value)).
		Update("expires_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	expired := requestJSON(t, router, http.MethodGet, "/api/v1/auth/me", nil, expiring)
	if expired.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status=%d body=%s", expired.Code, expired.Body.String())
	}
}
