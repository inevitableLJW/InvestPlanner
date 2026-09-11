package marketdata

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPurchaseRealUpstreamZeroLimitRows(t *testing.T) {
	// Values observed in EastMoney's fund purchase list on 2026-09-06.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"基金代码":"021778","申购状态":"限大额","购买起点":500,"日累计限定金额":0},{"基金代码":"021000","申购状态":"限大额","购买起点":10,"日累计限定金额":0},{"基金代码":"022525","申购状态":"限大额","购买起点":0.1,"日累计限定金额":0},{"基金代码":"000834","申购状态":"暂停申购","购买起点":10,"日累计限定金额":10}]`))
	}))
	defer server.Close()
	items, err := NewAKToolsClient(server.URL, time.Second).PurchaseLimits(context.Background())
	if err != nil || len(items) != 4 {
		t.Fatalf("%v %v", items, err)
	}
	for _, item := range items[:3] {
		if item.Status != "limited" || item.RawStatus != "限大额" || item.DailyLimitCents != nil {
			t.Fatalf("invalid semantics: %+v", item)
		}
	}
	if items[3].Status != "suspended" {
		t.Fatal(items[3])
	}
}

func TestPurchaseStatusAndNumericValidation(t *testing.T) {
	for raw, want := range map[string]string{"暂停大额申购": "limited", "限大额": "limited", "开放申购": "open", "暂停申购": "suspended", "开放赎回": "unknown", "--": "unknown"} {
		if got := normalizePurchaseStatus(raw); got != want {
			t.Errorf("%s: %s", raw, got)
		}
	}
	for _, raw := range []any{"NaN", "+Inf", "-Inf", "--", nil} {
		if parseFloatPtr(raw) != nil {
			t.Errorf("accepted %v", raw)
		}
	}
	for _, raw := range []any{-1, 1e100, "NaN"} {
		if parseCNYCents(raw) != nil {
			t.Errorf("accepted money %v", raw)
		}
	}
}

func TestETFInvalidPriceOrIOPVDoesNotGeneratePremium(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"代码":"513100","最新价":0,"IOPV":2},{"代码":"513500","最新价":2,"IOPV":0},{"代码":"159501","最新价":"NaN","IOPV":2}]`))
	}))
	defer server.Close()
	items, err := NewAKToolsClient(server.URL, time.Second).ETFQuotes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.PremiumRate != nil {
			t.Fatal(item)
		}
	}
}

func TestTrackingMatchesReturnIntervalsAndRejectsBadSeries(t *testing.T) {
	now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	base := make([]SeriesPoint, 150)
	for i := range base {
		base[i] = SeriesPoint{Date: now.AddDate(0, 0, i-149), Value: 100 * math.Pow(1.01, float64(i))}
	}
	fund := append([]SeriesPoint(nil), base[:70]...)
	fund = append(fund, base[71:]...)
	value, count, _, reason := AnnualizedTrackingError(fund, "adjusted_nav", base, "total_return", now)
	if reason != "" || count != 147 || math.Abs(value) > 1e-12 {
		t.Fatalf("%g %d %s", value, count, reason)
	}
	withOldAdjustment := append([]SeriesPoint{{Date: time.Date(2002, 7, 2, 0, 0, 0, 0, time.UTC), Value: -0.4883}}, base...)
	if _, _, _, reason := AnnualizedTrackingError(fund, "adjusted_nav", withOldAdjustment, "total_return", now); reason != "" {
		t.Fatal("out-of-window additive adjustment blocked current window", reason)
	}
	for _, bad := range []SeriesPoint{{Date: now.AddDate(0, 0, 1), Value: 100}, {Date: now, Value: math.NaN()}, {Date: now, Value: 0}} {
		damaged := append([]SeriesPoint(nil), base...)
		damaged[len(damaged)-1] = bad
		if _, _, _, reason := AnnualizedTrackingError(damaged, "adjusted_nav", base, "total_return", now); reason == "" {
			t.Fatal("accepted invalid series")
		}
	}
}

func TestUSBenchmarkIgnoresAncientAdjustmentButRejectsCurrentInvalidPrice(t *testing.T) {
	badCurrent := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if badCurrent {
			_, _ = w.Write([]byte(`[{"date":"2026-09-04","close":-1}]`))
			return
		}
		_, _ = w.Write([]byte(`[{"date":"2002-07-02","close":-0.4883},{"date":"2026-09-03","close":717.67},{"date":"2026-09-04","close":718.96}]`))
	}))
	defer upstream.Close()
	c := NewAKToolsClient(upstream.URL, time.Second)
	c.Now = func() time.Time { return time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC) }
	series, err := c.USFundAdjustedClose(context.Background(), "QQQ")
	if err != nil || len(series.Points) != 2 {
		t.Fatalf("%+v %v", series, err)
	}
	badCurrent = true
	if _, err := c.USFundAdjustedClose(context.Background(), "QQQ"); err == nil {
		t.Fatal("accepted bad current price")
	}
}
