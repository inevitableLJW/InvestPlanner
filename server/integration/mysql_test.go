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
		"qdii_assistant_confirmations",
		"market_data_snapshots",
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
	store := repository.New(db)
	app := service.New(store, time.Hour)
	app.QDII = &service.QDIIService{Store: store, Now: time.Now, FreshFor: 24 * time.Hour}
	app.FX = &service.FXService{Store: store, Now: time.Now, Enabled: true}
	return httpapi.New(cfg, app)
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
	destinations := plan["destinations"].([]any)
	if cash := destinations[0].(map[string]any); cash["executionMode"] != "local" || cash["settlementCurrency"] != "CNY" || cash["fxQuotaRelevant"] != false {
		t.Fatalf("cash preset profile=%#v", cash)
	}
	if qdii := destinations[2].(map[string]any); qdii["executionMode"] != "qdii" || qdii["settlementCurrency"] != "CNY" || qdii["benchmarkId"] != "NASDAQ100_TR" {
		t.Fatalf("QDII preset profile=%#v", qdii)
	}

	for _, raw := range destinations {
		destination := raw.(map[string]any)
		destination["executionMode"] = "direct_us"
		destination["settlementCurrency"] = "USD"
		destination["benchmarkId"] = "wrong"
		destination["fxQuotaRelevant"] = true
	}
	updated := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+planID, map[string]any{
		"name": "长期计划", "action": "save_draft", "defaultContributionBps": 8000,
		"reserveCents": 0, "roundingUnitCents": 10000, "version": plan["version"], "destinations": destinations,
	}, first)
	if updated.Code != http.StatusOK {
		t.Fatalf("update plan status=%d body=%s", updated.Code, updated.Body.String())
	}
	var updatedPlan map[string]any
	if err := json.Unmarshal(updated.Body.Bytes(), &updatedPlan); err != nil {
		t.Fatal(err)
	}
	updatedCash := updatedPlan["destinations"].([]any)[0].(map[string]any)
	if updatedCash["executionMode"] != "local" || updatedCash["settlementCurrency"] != "CNY" || updatedCash["benchmarkId"] != "" || updatedCash["fxQuotaRelevant"] != false {
		t.Fatalf("server did not enforce preset profile: %#v", updatedCash)
	}

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

func TestQDIIConfirmationPersistsForTheSamePlanMonthAndDestination(t *testing.T) {
	router := testRouter(t)
	cookie := register(t, router, "qdii-confirm-user")
	created := requestJSON(t, router, http.MethodPost, "/api/v1/plans", map[string]any{"name": "QDII 计划"}, cookie)
	var plan map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	destinations := plan["destinations"].([]any)
	qdii := destinations[2].(map[string]any)
	for _, raw := range destinations {
		destination := raw.(map[string]any)
		destination["active"] = destination["id"] == qdii["id"]
		destination["allocationBps"] = 0
	}
	qdii["allocationBps"] = 10000
	updated := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+plan["id"].(string), map[string]any{
		"name": "QDII 计划", "action": "publish", "defaultContributionBps": 10000,
		"reserveCents": 0, "roundingUnitCents": 10000, "version": plan["version"], "destinations": destinations,
	}, cookie)
	if updated.Code != http.StatusOK {
		t.Fatalf("publish status=%d body=%s", updated.Code, updated.Body.String())
	}
	month := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+plan["id"].(string)+"/months/2026-08", map[string]any{
		"incomeCents": 100000, "contributionBps": 10000, "expenses": []any{}, "note": "",
	}, cookie)
	if month.Code != http.StatusOK {
		t.Fatalf("month status=%d body=%s", month.Code, month.Body.String())
	}
	path := "/api/v1/plans/" + plan["id"].(string) + "/assistants/qdii/" + qdii["id"].(string)
	confirmed := requestJSON(t, router, http.MethodPut, path, map[string]any{
		"month": "2026-08", "selectedOtcFundCodes": []string{}, "selectedEtfFundCode": "513100",
	}, cookie)
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirm status=%d body=%s", confirmed.Code, confirmed.Body.String())
	}
	db, err := database.Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	var firstConfirmation database.QDIIAssistantConfirmation
	if err := db.Where("destination_id = ?", qdii["id"].(string)).First(&firstConfirmation).Error; err != nil {
		t.Fatal(err)
	}
	reconfirmed := requestJSON(t, router, http.MethodPut, path, map[string]any{
		"month": "2026-08", "selectedOtcFundCodes": []string{}, "selectedEtfFundCode": "513100",
	}, cookie)
	if reconfirmed.Code != http.StatusOK {
		t.Fatalf("reconfirm status=%d body=%s", reconfirmed.Code, reconfirmed.Body.String())
	}
	var updatedConfirmation database.QDIIAssistantConfirmation
	if err := db.Where("destination_id = ?", qdii["id"].(string)).First(&updatedConfirmation).Error; err != nil {
		t.Fatal(err)
	}
	if updatedConfirmation.ID != firstConfirmation.ID {
		t.Fatalf("same-month confirmation id changed: before=%s after=%s", firstConfirmation.ID, updatedConfirmation.ID)
	}
	if updatedConfirmation.SelectedETFAmountCents == nil || *updatedConfirmation.SelectedETFAmountCents != 100000 {
		t.Fatalf("selected ETF amount=%v, want 100000", updatedConfirmation.SelectedETFAmountCents)
	}
	var confirmationCount int64
	if err := db.Model(&database.QDIIAssistantConfirmation{}).Where("destination_id = ?", qdii["id"].(string)).Count(&confirmationCount).Error; err != nil {
		t.Fatal(err)
	}
	if confirmationCount != 1 {
		t.Fatalf("same-month confirmation count=%d, want 1", confirmationCount)
	}
	loaded := requestJSON(t, router, http.MethodGet, path+"?remainingNeedCents=100000&month=2026-08", nil, cookie)
	if loaded.Code != http.StatusOK || !bytes.Contains(loaded.Body.Bytes(), []byte(`"selectedOtcFundCodes":[]`)) || !bytes.Contains(loaded.Body.Bytes(), []byte(`"selectedOtcPlans":[]`)) || !bytes.Contains(loaded.Body.Bytes(), []byte(`"selectedEtfFundCode":"513100"`)) || !bytes.Contains(loaded.Body.Bytes(), []byte(`"selectedEtfAmountCents":100000`)) {
		t.Fatalf("load status=%d body=%s", loaded.Code, loaded.Body.String())
	}
	summaries := requestJSON(t, router, http.MethodGet, "/api/v1/plans/"+plan["id"].(string)+"/months/2026-08/qdii-confirmations", nil, cookie)
	if summaries.Code != http.StatusOK || !bytes.Contains(summaries.Body.Bytes(), []byte(`"destinationId":"`+qdii["id"].(string)+`"`)) || !bytes.Contains(summaries.Body.Bytes(), []byte(`"fundCode":"513100"`)) || !bytes.Contains(summaries.Body.Bytes(), []byte(`"selectedEtfAmountCents":100000`)) {
		t.Fatalf("summary status=%d body=%s", summaries.Code, summaries.Body.String())
	}
}

func TestFXForecastReturnsCNYDemandWithoutRateSnapshots(t *testing.T) {
	router := testRouter(t)
	cookie := register(t, router, "fx-default-user")
	created := requestJSON(t, router, http.MethodPost, "/api/v1/plans", map[string]any{"name": "美股计划"}, cookie)
	var plan map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	destinations := []map[string]any{{
		"name": "苹果", "active": true, "allocationBps": 10000, "sortOrder": 0,
		"executionMode": "direct_us", "settlementCurrency": "USD", "benchmarkId": "", "fxQuotaRelevant": true,
	}}
	updated := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+plan["id"].(string), map[string]any{
		"name": "美股计划", "action": "publish", "defaultContributionBps": 10000,
		"reserveCents": 0, "roundingUnitCents": 10000, "version": plan["version"], "destinations": destinations,
	}, cookie)
	if updated.Code != http.StatusOK {
		t.Fatalf("publish status=%d body=%s", updated.Code, updated.Body.String())
	}
	month := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+plan["id"].(string)+"/months/2026-08", map[string]any{
		"incomeCents": 100000, "contributionBps": 10000, "expenses": []any{}, "note": "",
	}, cookie)
	if month.Code != http.StatusOK {
		t.Fatalf("month status=%d body=%s", month.Code, month.Body.String())
	}
	forecast := requestJSON(t, router, http.MethodGet, "/api/v1/plans/"+plan["id"].(string)+"/assistants/fx?window=rolling_12", nil, cookie)
	if forecast.Code != http.StatusOK {
		t.Fatalf("forecast status=%d body=%s", forecast.Code, forecast.Body.String())
	}
	var payload service.FXForecast
	if err := json.Unmarshal(forecast.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "fresh" || len(payload.Currencies) != 1 || payload.Currencies[0] != "USD" || len(payload.Summaries) != 1 || payload.Summaries[0].CNYCents == nil || *payload.Summaries[0].CNYCents == 0 || payload.Summaries[0].ForeignAmount != nil {
		t.Fatalf("forecast without rates=%s", forecast.Body.String())
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
	legacy := register(t, router, "legacy-session-user")
	if err := db.Model(&database.Session{}).
		Where("token_hash = ?", service.HashToken(legacy.Value)).
		Updates(map[string]any{
			"created_at": time.Now().Add(-2 * time.Hour),
			"expires_at": time.Now().Add(24 * time.Hour),
		}).Error; err != nil {
		t.Fatal(err)
	}
	shortened := requestJSON(t, router, http.MethodGet, "/api/v1/auth/me", nil, legacy)
	if shortened.Code != http.StatusUnauthorized {
		t.Fatalf("shortened session status=%d body=%s", shortened.Code, shortened.Body.String())
	}
}
