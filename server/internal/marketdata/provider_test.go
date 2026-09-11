package marketdata

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAKToolsCatalogAndFieldContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("symbol") != "QDII" {
			t.Errorf("symbol=%q", r.URL.Query().Get("symbol"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"基金代码":"000001","基金简称":"示例基金A"}]`))
	}))
	defer server.Close()
	client := NewAKToolsClient(server.URL, time.Second)
	items, err := client.DiscoverQDII(context.Background())
	if err != nil || len(items) != 1 || items[0].ShareClass != "A" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`[{"代码":"000001"}]`)) })
	if _, err := client.DiscoverQDII(context.Background()); err == nil {
		t.Fatal("expected field rename rejection")
	}
}

func TestFundCumulativeNAVRetriesRepeatedTransientFailures(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 6 {
			http.Error(w, "temporary upstream failure", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`[{"净值日期":"2026-09-03T00:00:00.000","累计净值":1.20},{"净值日期":"2026-09-04T00:00:00.000","累计净值":1.21}]`))
	}))
	defer server.Close()

	client := NewAKToolsClient(server.URL, 5*time.Second)
	series, err := client.FundCumulativeNAV(context.Background(), "513100")
	if err != nil || calls != 6 || series.Basis != "cumulative_nav_proxy" || len(series.Points) != 2 {
		t.Fatalf("calls=%d series=%+v err=%v", calls, series, err)
	}
}

func TestFundCumulativeNAVRetriesTemporaryEmptyResponses(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`[{"净值日期":"2026-09-03T00:00:00.000","累计净值":1.20},{"净值日期":"2026-09-04T00:00:00.000","累计净值":1.21}]`))
	}))
	defer server.Close()

	client := NewAKToolsClient(server.URL, 5*time.Second)
	series, err := client.FundCumulativeNAV(context.Background(), "513100")
	if err != nil || calls != 3 || len(series.Points) != 2 {
		t.Fatalf("calls=%d series=%+v err=%v", calls, series, err)
	}
}

func TestFundCumulativeNAVSkipsNullObservationsWithoutZeroFilling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"净值日期":"2020-12-30T00:00:00.000","累计净值":1.20},{"净值日期":"2020-12-31T00:00:00.000","累计净值":null},{"净值日期":"2021-01-04T00:00:00.000","累计净值":1.21}]`))
	}))
	defer server.Close()

	series, err := NewAKToolsClient(server.URL, time.Second).FundCumulativeNAV(context.Background(), "513100")
	if err != nil || len(series.Points) != 2 {
		t.Fatalf("series=%+v err=%v", series, err)
	}
	if series.Points[0].Value != 1.20 || series.Points[1].Value != 1.21 {
		t.Fatalf("missing observation was not omitted: %+v", series.Points)
	}
}

func TestAKToolsETFUsesRealNameAndIOPVFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"代码":"513100","名称":"纳指ETF国泰","最新价":3,"IOPV实时估值":2,"基金折价率":-50}]`))
	}))
	defer server.Close()
	client := NewAKToolsClient(server.URL, time.Second)
	items, err := client.ETFQuotes(context.Background())
	if err != nil || len(items) != 1 || items[0].Name != "纳指ETF国泰" || items[0].IOPV == nil || *items[0].IOPV != 2 || items[0].PremiumRate == nil || *items[0].PremiumRate != .5 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestAKToolsRejectsEmptyAndBadUnits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`[]`)) }))
	defer server.Close()
	client := NewAKToolsClient(server.URL, time.Second)
	if _, err := client.PurchaseLimits(context.Background()); err == nil {
		t.Fatal("expected empty rejection")
	}
	if parseCNYCents("not-money") != nil {
		t.Fatal("unknown amount must stay nil")
	}
}

func TestAKToolsPurchaseUsesPinnedRealFieldNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"基金代码":"006479","基金简称":"广发纳斯达克100ETF联接人民币(QDII)C","申购状态":"限大额","购买起点":5,"日累计限定金额":5}]`))
	}))
	defer server.Close()
	client := NewAKToolsClient(server.URL, time.Second)
	items, err := client.PurchaseLimits(context.Background())
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].Name == "" || items[0].ShareClass != "C" || items[0].Venue != "otc" || items[0].Status != "limited" || items[0].MinimumCents == nil || *items[0].MinimumCents != 500 || items[0].DailyLimitCents == nil || *items[0].DailyLimitCents != 500 {
		t.Fatalf("unexpected normalized purchase: %+v", items[0])
	}
}

func TestAKToolsFreeTrackingProxyContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/api/public/") {
		case "fund_open_fund_info_em":
			if r.URL.Query().Get("indicator") != "累计净值走势" {
				t.Fatalf("indicator=%q", r.URL.Query().Get("indicator"))
			}
			_, _ = w.Write([]byte(`[{"净值日期":"2026-08-20T00:00:00.000","累计净值":2.0},{"净值日期":"2026-08-21T00:00:00.000","累计净值":2.1}]`))
		case "fund_etf_hist_em":
			query := r.URL.Query()
			if query.Get("symbol") != "159501" || query.Get("period") != "daily" || query.Get("adjust") != "qfq" || query.Get("start_date") != "20250722" || query.Get("end_date") != "20260821" {
				t.Fatalf("query=%v", query)
			}
			_, _ = w.Write([]byte(`[{"日期":"2026-08-20T00:00:00.000","收盘":1.2},{"日期":"2026-08-21T00:00:00.000","收盘":1.21}]`))
		case "fund_etf_fund_info_em":
			query := r.URL.Query()
			if query.Get("fund") != "159501" || query.Get("start_date") != "20250722" || query.Get("end_date") != "20260821" {
				t.Fatalf("query=%v", query)
			}
			_, _ = w.Write([]byte(`[{"净值日期":"2026-08-20T00:00:00.000","累计净值":1.2},{"净值日期":"2026-08-21T00:00:00.000","累计净值":1.21}]`))
		case "stock_us_daily":
			if r.URL.Query().Get("symbol") != "QQQ" || r.URL.Query().Get("adjust") != "qfq" {
				t.Fatalf("query=%v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`[{"date":"2026-08-20T00:00:00.000","close":100},{"date":"2026-08-21T00:00:00.000","close":101}]`))
		case "currency_boc_safe":
			_, _ = w.Write([]byte(`[{"日期":"2026-08-20T00:00:00.000","美元":700},{"日期":"2026-08-21T00:00:00.000","美元":701}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := NewAKToolsClient(server.URL, time.Second)
	client.Now = func() time.Time { return time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC) }
	fund, err := client.FundCumulativeNAV(context.Background(), "000001")
	if err != nil || fund.Basis != "cumulative_nav_proxy" || len(fund.Points) != 2 {
		t.Fatalf("fund=%+v err=%v", fund, err)
	}
	domesticETF, err := client.FundETFAdjustedClose(context.Background(), "159501")
	if err != nil || domesticETF.Basis != "adjusted_close_cny" || len(domesticETF.Points) != 2 {
		t.Fatalf("domestic ETF=%+v err=%v", domesticETF, err)
	}
	etfNAV, err := client.FundETFCumulativeNAV(context.Background(), "159501")
	if err != nil || etfNAV.Basis != "cumulative_nav_proxy" || len(etfNAV.Points) != 2 {
		t.Fatalf("ETF NAV=%+v err=%v", etfNAV, err)
	}
	etf, err := client.USFundAdjustedClose(context.Background(), "QQQ")
	if err != nil {
		t.Fatal(err)
	}
	fx, err := client.USDCNYReferenceHistory(context.Background())
	if err != nil || fx.Points[0].Value != 7 {
		t.Fatalf("fx=%+v err=%v", fx, err)
	}
	combined, err := CNYAdjustedBenchmark(etf, fx)
	if err != nil || combined.Basis != "adjusted_close_cny_proxy" || combined.Points[1].Value != 708.01 {
		t.Fatalf("combined=%+v err=%v", combined, err)
	}
}

func TestAKToolsSAFEReferenceRatesUseLatestUSDAndHKD(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/public/currency_boc_safe") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
			{"日期":"2026-08-20T00:00:00.000","美元":700,"港元":90},
			{"日期":"2026-08-21T00:00:00.000","美元":701,"港元":91.23}
		]`))
	}))
	defer server.Close()
	client := NewAKToolsClient(server.URL, time.Second)
	client.Now = func() time.Time { return time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC) }
	rates, err := client.SAFEReferenceRates(context.Background())
	if err != nil || len(rates) != 2 {
		t.Fatalf("rates=%+v err=%v", rates, err)
	}
	if rates[0].Currency != "USD" || rates[0].CNYPerUnit != 7.01 || rates[1].Currency != "HKD" || rates[1].CNYPerUnit != .9123 {
		t.Fatalf("unexpected rates: %+v", rates)
	}
	for _, rate := range rates {
		if rate.Meta.Provider != "safe" || rate.Meta.SourceURL != SAFEReferenceRateSource || rate.Meta.ObservedAt.Format("2006-01-02") != "2026-08-21" {
			t.Fatalf("unexpected metadata: %+v", rate.Meta)
		}
	}
}

func TestAKToolsETFTrackingFallbackRequiresNoCorporateActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/api/public/") {
		case "fund_etf_hist_sina":
			if r.URL.Query().Get("symbol") != "sz159501" {
				t.Fatalf("symbol=%q", r.URL.Query().Get("symbol"))
			}
			_, _ = w.Write([]byte(`[{"date":"2026-08-20T00:00:00.000","close":1.2},{"date":"2026-08-21T00:00:00.000","close":1.21}]`))
		case "fund_etf_dividend_sina":
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := NewAKToolsClient(server.URL, time.Second)
	client.Now = func() time.Time { return time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC) }
	series, err := client.FundETFUnadjustedCloseNoActions(context.Background(), "159501")
	if err != nil || series.Basis != "close_cny_no_corporate_action" || len(series.Points) != 2 {
		t.Fatalf("series=%+v err=%v", series, err)
	}
}

func TestExactBenchmarkMappingUsesOverviewTargetNotName(t *testing.T) {
	if id, ok := exactBenchmarkID("纳斯达克100指数"); !ok || id != "NASDAQ100_TR" {
		t.Fatalf("id=%q ok=%v", id, ok)
	}
	if id, ok := exactBenchmarkID("标准普尔500指数"); !ok || id != "SP500_TR" {
		t.Fatalf("id=%q ok=%v", id, ok)
	}
	for _, target := range []string{"该基金无跟踪标的", "纳斯达克科技市值加权指数", "纳斯达克生物科技指数", "标普500等权重指数"} {
		if id, ok := exactBenchmarkID(target); ok {
			t.Fatalf("target=%q must not map, got %q", target, id)
		}
	}
	if !isRMBShare("摩根纳斯达克100指数(QDII)人民币A") || isRMBShare("摩根纳斯达克100指数(QDII)美元现汇A") {
		t.Fatal("RMB share classification failed")
	}
}

func TestChinaMoneyJSONPAndUnit(t *testing.T) {
	rows, err := parseChinaMoney([]byte(`callback({"records":[{"currency":"HKD","date":"2026-08-21","value":91.23,"unit":"CNY/100"}]})`))
	if err != nil || len(rows) != 1 {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`cb({"records":[{"currency":"HKD","date":"2026-08-21","value":91.23,"unit":"CNY/100"}]})`))
	}))
	defer server.Close()
	client := NewChinaMoneyClient(server.URL, time.Second)
	rates, err := client.ReferenceRates(context.Background(), time.Now())
	if err != nil || rates[0].CNYPerUnit != 0.9123 {
		t.Fatalf("rates=%+v err=%v", rates, err)
	}
	if _, err := parseChinaMoney([]byte(`cb({"records":[{"currency":"USD"}]})`)); err == nil {
		t.Fatal("expected contract rejection")
	}
}

func TestTushareArrayPositionsAndAdjustedNAV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"fund_nav"`) {
			t.Fatal("wrong API")
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"fields":["ts_code","nav_date","adj_nav"],"items":[["000001.OF","20260821",1.25]]}}`))
	}))
	defer server.Close()
	client := NewTushareClient("secret", time.Second)
	client.Endpoint = server.URL
	series, err := client.FundNAV(context.Background(), "000001.OF")
	if err != nil || series.Basis != "adjusted_nav" || series.Points[0].Value != 1.25 {
		t.Fatalf("series=%+v err=%v", series, err)
	}
}

func TestTushareRejectsRenamedAndShortArrays(t *testing.T) {
	response := tushareResponse{}
	response.Data.Fields = []string{"date", "value"}
	response.Data.Items = [][]any{{"20260821"}}
	if _, err := tusharePoints(response, "nav_date", "adj_nav"); err == nil {
		t.Fatal("expected renamed fields rejection")
	}
	response.Data.Fields = []string{"nav_date", "adj_nav"}
	if _, err := tusharePoints(response, "nav_date", "adj_nav"); err == nil {
		t.Fatal("expected short row rejection")
	}
}

func TestClassifyAndCrossValidate(t *testing.T) {
	now := time.Now()
	if Classify(now.Add(-2*time.Hour), now, time.Hour) != Stale {
		t.Fatal("expected stale")
	}
	a, b := 1.0, 1.01
	if _, source, err := CrossValidate(&a, &b, .02); err != nil || source != "cross_validated" {
		t.Fatal(source, err)
	}
	b = 1.2
	if _, source, err := CrossValidate(&a, &b, .02); err == nil || source != "snapshot" {
		t.Fatal("expected snapshot fallback")
	}
}
