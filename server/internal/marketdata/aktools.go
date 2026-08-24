package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type AKToolsClient struct {
	BaseURL string
	HTTP    *http.Client
	Now     func() time.Time
}

func NewAKToolsClient(baseURL string, timeout time.Duration) *AKToolsClient {
	return &AKToolsClient{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: timeout}, Now: time.Now}
}

func (c *AKToolsClient) rows(ctx context.Context, fn string, query url.Values) ([]map[string]any, string, error) {
	rows, source, err := c.rowsAllowEmpty(ctx, fn, query)
	if err != nil {
		return nil, source, err
	}
	if len(rows) == 0 {
		return nil, source, fmt.Errorf("aktools empty response")
	}
	return rows, source, nil
}

func (c *AKToolsClient) rowsAllowEmpty(ctx context.Context, fn string, query url.Values) ([]map[string]any, string, error) {
	source := c.BaseURL + "/api/public/" + fn
	if encoded := query.Encode(); encoded != "" {
		source += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, source, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, source, fmt.Errorf("aktools request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, source, fmt.Errorf("aktools status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, source, fmt.Errorf("aktools read failed")
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, source, fmt.Errorf("aktools invalid json")
	}
	return rows, source, nil
}

func (c *AKToolsClient) DiscoverQDII(ctx context.Context) ([]FundCatalogItem, error) {
	rows, source, err := c.rows(ctx, "fund_open_fund_rank_em", url.Values{"symbol": {"QDII"}})
	if err != nil {
		return nil, err
	}
	result := make([]FundCatalogItem, 0, len(rows))
	now := c.Now().UTC()
	for _, row := range rows {
		code, ok := textField(row, "基金代码", "基金简称")
		if !ok {
			return nil, fmt.Errorf("aktools catalog contract changed")
		}
		name, _ := valueString(row["基金简称"])
		result = append(result, FundCatalogItem{Code: code, Name: name, ShareClass: inferShareClass(name), Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: now, FetchedAt: now}})
	}
	return result, nil
}

func (c *AKToolsClient) PurchaseLimits(ctx context.Context) ([]PurchaseLimit, error) {
	rows, source, err := c.rows(ctx, "fund_purchase_em", nil)
	if err != nil {
		return nil, err
	}
	now := c.Now().UTC()
	result := make([]PurchaseLimit, 0, len(rows))
	for _, row := range rows {
		code, ok := textField(row, "基金代码")
		if !ok {
			return nil, fmt.Errorf("aktools purchase contract changed")
		}
		name, _ := textField(row, "基金简称", "基金名称")
		status, _ := valueString(first(row, "申购状态", "购买状态"))
		venue := "otc"
		if strings.Contains(status, "场内") {
			venue = "etf"
		}
		nextOpen := parseOptionalDate(first(row, "下一开放日", "开放日"))
		minimum := parseCNYCents(first(row, "购买起点", "起购金额", "最低购买额"))
		limit := parseCNYCents(first(row, "日累计限定金额", "日累计限额", "日限额"))
		result = append(result, PurchaseLimit{FundCode: code, Name: name, ShareClass: inferShareClass(name), Venue: venue, Channel: "天天基金", Status: normalizePurchaseStatus(status), NextOpenDate: nextOpen, MinimumCents: minimum, DailyLimitCents: limit, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: now, FetchedAt: now}})
	}
	return result, nil
}

func (c *AKToolsClient) Overview(ctx context.Context, code string) (FundOverview, error) {
	rows, source, err := c.rows(ctx, "fund_overview_em", url.Values{"symbol": {code}})
	if err != nil {
		return FundOverview{}, err
	}
	now := c.Now().UTC()
	row := rows[0]
	return FundOverview{FundCode: code, TrackingTarget: anyText(first(row, "跟踪标的", "跟踪指数")), PerformanceBenchmark: anyText(first(row, "业绩比较基准")), Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: now, FetchedAt: now}}, nil
}

func (c *AKToolsClient) FundNAV(ctx context.Context, code string) (NAVSeries, error) {
	rows, source, err := c.rows(ctx, "fund_open_fund_info_em", url.Values{"symbol": {code}, "indicator": {"单位净值走势"}})
	if err != nil {
		return NAVSeries{}, err
	}
	points, err := parsePoints(rows, []string{"净值日期", "日期"}, []string{"单位净值", "净值"})
	if err != nil {
		return NAVSeries{}, err
	}
	return NAVSeries{FundCode: code, Basis: "unit_nav", Points: points, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: points[len(points)-1].Date, FetchedAt: c.Now().UTC()}}, nil
}

// FundCumulativeNAV returns EastMoney's cumulative NAV curve. It is kept under
// an explicit proxy basis because cumulative NAV is not the same contractual
// return basis as an authorized adjusted NAV series.
func (c *AKToolsClient) FundCumulativeNAV(ctx context.Context, code string) (NAVSeries, error) {
	rows, source, err := c.rows(ctx, "fund_open_fund_info_em", url.Values{"symbol": {code}, "indicator": {"累计净值走势"}})
	if err != nil {
		return NAVSeries{}, err
	}
	points, err := parsePoints(rows, []string{"净值日期", "日期"}, []string{"累计净值", "净值"})
	if err != nil {
		return NAVSeries{}, err
	}
	return NAVSeries{FundCode: code, Basis: "cumulative_nav_proxy", Points: points, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: points[len(points)-1].Date, FetchedAt: c.Now().UTC()}}, nil
}

// FundETFAdjustedClose returns a domestic ETF's CNY-denominated, forward-adjusted
// daily close series. It is compared with the CNY-converted QQQ/SPY series.
func (c *AKToolsClient) FundETFAdjustedClose(ctx context.Context, code string) (NAVSeries, error) {
	now := c.Now().UTC()
	rows, source, err := c.rows(ctx, "fund_etf_hist_em", url.Values{
		"symbol":     {code},
		"period":     {"daily"},
		"start_date": {now.AddDate(-1, 0, -30).Format("20060102")},
		"end_date":   {now.Format("20060102")},
		"adjust":     {"qfq"},
	})
	if err != nil {
		return NAVSeries{}, err
	}
	points, err := parsePoints(rows, []string{"日期", "date"}, []string{"收盘", "close"})
	if err != nil {
		return NAVSeries{}, err
	}
	return NAVSeries{FundCode: code, Basis: "adjusted_close_cny", Points: points, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: points[len(points)-1].Date, FetchedAt: now}}, nil
}

// FundETFCumulativeNAV returns the exchange-traded fund's published cumulative
// NAV. It is preferred over market prices for conventional fund tracking error.
func (c *AKToolsClient) FundETFCumulativeNAV(ctx context.Context, code string) (NAVSeries, error) {
	now := c.Now().UTC()
	rows, source, err := c.rows(ctx, "fund_etf_fund_info_em", url.Values{
		"fund":       {code},
		"start_date": {now.AddDate(-1, 0, -30).Format("20060102")},
		"end_date":   {now.Format("20060102")},
	})
	if err != nil {
		return NAVSeries{}, err
	}
	points, err := parsePoints(rows, []string{"净值日期", "日期"}, []string{"累计净值"})
	if err != nil {
		return NAVSeries{}, err
	}
	return NAVSeries{FundCode: code, Basis: "cumulative_nav_proxy", Points: points, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: points[len(points)-1].Date, FetchedAt: now}}, nil
}

// FundETFUnadjustedCloseNoActions is a controlled fallback for outages of the
// adjusted EastMoney series. Raw Sina closes are accepted only when the last
// year has no reported cash dividend and no scale jump that could be a split.
func (c *AKToolsClient) FundETFUnadjustedCloseNoActions(ctx context.Context, code string) (NAVSeries, error) {
	now := c.Now().UTC()
	prefix := "sz"
	if strings.HasPrefix(code, "5") {
		prefix = "sh"
	}
	symbol := prefix + code
	rows, source, err := c.rows(ctx, "fund_etf_hist_sina", url.Values{"symbol": {symbol}})
	if err != nil {
		return NAVSeries{}, err
	}
	points, err := parsePoints(rows, []string{"date", "日期"}, []string{"close", "收盘"})
	if err != nil {
		return NAVSeries{}, err
	}
	cutoff := now.AddDate(-1, 0, -30)
	dividends, dividendSource, err := c.rowsAllowEmpty(ctx, "fund_etf_dividend_sina", url.Values{"symbol": {symbol}})
	if err != nil {
		return NAVSeries{}, err
	}
	for _, row := range dividends {
		if date := parseOptionalDate(first(row, "日期", "date")); date != nil && !date.Before(cutoff) {
			return NAVSeries{}, fmt.Errorf("ETF fallback series has a recent dividend")
		}
	}
	for i := 1; i < len(points); i++ {
		if points[i].Date.Before(cutoff) || points[i-1].Value <= 0 {
			continue
		}
		ratio := points[i].Value / points[i-1].Value
		if ratio < 0.67 || ratio > 1.5 {
			return NAVSeries{}, fmt.Errorf("ETF fallback series has a possible scale change")
		}
	}
	return NAVSeries{FundCode: code, Basis: "close_cny_no_corporate_action", Points: points, Meta: SourceMeta{Provider: "aktools-sina", SourceURL: source + " | " + dividendSource, ObservedAt: points[len(points)-1].Date, FetchedAt: now}}, nil
}

// USFundAdjustedClose uses the free AKShare/Sina adjusted daily series for the
// comparison ETF selected by the product (QQQ or SPY).
func (c *AKToolsClient) USFundAdjustedClose(ctx context.Context, symbol string) (BenchmarkSeries, error) {
	rows, source, err := c.rows(ctx, "stock_us_daily", url.Values{"symbol": {symbol}, "adjust": {"qfq"}})
	if err != nil {
		return BenchmarkSeries{}, err
	}
	points, err := parsePoints(rows, []string{"date", "日期"}, []string{"close", "收盘"})
	if err != nil {
		return BenchmarkSeries{}, err
	}
	return BenchmarkSeries{BenchmarkID: symbol, Basis: "adjusted_close_usd_proxy", Points: points, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: points[len(points)-1].Date, FetchedAt: c.Now().UTC()}}, nil
}

// USDCNYReferenceHistory reads the free SAFE central-parity history exposed by
// AKShare. SAFE publishes USD as CNY per 100 USD, so values are normalized to
// CNY per USD before being combined with QQQ/SPY.
func (c *AKToolsClient) USDCNYReferenceHistory(ctx context.Context) (BenchmarkSeries, error) {
	rows, source, err := c.rows(ctx, "currency_boc_safe", nil)
	if err != nil {
		return BenchmarkSeries{}, err
	}
	points, err := parsePoints(rows, []string{"日期", "date"}, []string{"美元", "USD"})
	if err != nil {
		return BenchmarkSeries{}, err
	}
	for i := range points {
		points[i].Value /= 100
	}
	return BenchmarkSeries{BenchmarkID: "USDCNY", Basis: "safe_middle_rate", Points: points, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: points[len(points)-1].Date, FetchedAt: c.Now().UTC()}}, nil
}

func CNYAdjustedBenchmark(etf, fx BenchmarkSeries) (BenchmarkSeries, error) {
	if etf.Basis != "adjusted_close_usd_proxy" || fx.Basis != "safe_middle_rate" {
		return BenchmarkSeries{}, fmt.Errorf("proxy benchmark basis incompatible")
	}
	fxByDate := make(map[time.Time]float64, len(fx.Points))
	for _, point := range fx.Points {
		fxByDate[point.Date] = point.Value
	}
	points := make([]SeriesPoint, 0, len(etf.Points))
	for _, point := range etf.Points {
		rate := 0.0
		for lag := 0; lag <= 7; lag++ {
			if value, ok := fxByDate[point.Date.AddDate(0, 0, -lag)]; ok {
				rate = value
				break
			}
		}
		if rate > 0 {
			points = append(points, SeriesPoint{Date: point.Date, Value: point.Value * rate})
		}
	}
	if len(points) == 0 {
		return BenchmarkSeries{}, fmt.Errorf("proxy benchmark has no aligned FX observations")
	}
	observedAt := points[len(points)-1].Date
	return BenchmarkSeries{
		BenchmarkID: etf.BenchmarkID,
		Basis:       "adjusted_close_cny_proxy",
		Points:      points,
		Meta: SourceMeta{
			Provider:   etf.Meta.Provider + "+safe",
			SourceURL:  etf.Meta.SourceURL + " | " + fx.Meta.SourceURL,
			ObservedAt: observedAt,
			FetchedAt:  etf.Meta.FetchedAt,
		},
	}, nil
}

func (c *AKToolsClient) ETFQuotes(ctx context.Context) ([]ETFQuote, error) {
	rows, source, err := c.rows(ctx, "fund_etf_spot_em", nil)
	if err != nil {
		return nil, err
	}
	now := c.Now().UTC()
	result := make([]ETFQuote, 0, len(rows))
	for _, row := range rows {
		code, ok := textField(row, "代码", "基金代码")
		if !ok {
			return nil, fmt.Errorf("aktools etf contract changed")
		}
		name, _ := textField(row, "名称", "基金简称")
		price := parseFloatPtr(first(row, "最新价", "最新"))
		iopv := parseFloatPtr(first(row, "IOPV实时估值", "IOPV"))
		var premiumRate *float64
		if price != nil && iopv != nil && *iopv > 0 {
			value := *price/(*iopv) - 1
			premiumRate = &value
		}
		result = append(result, ETFQuote{FundCode: code, Name: name, Price: price, IOPV: iopv, PremiumRate: premiumRate, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: now, FetchedAt: now}})
	}
	return result, nil
}

func (c *AKToolsClient) Benchmark(ctx context.Context, id string, from, to time.Time) (BenchmarkSeries, error) {
	rows, source, err := c.rows(ctx, "index_us_stock_sina", url.Values{"symbol": {id}})
	if err != nil {
		return BenchmarkSeries{}, err
	}
	points, err := parsePoints(rows, []string{"date", "日期"}, []string{"close", "收盘"})
	if err != nil {
		return BenchmarkSeries{}, err
	}
	filtered := points[:0]
	for _, p := range points {
		if !p.Date.Before(from) && !p.Date.After(to) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return BenchmarkSeries{}, fmt.Errorf("aktools benchmark empty range")
	}
	return BenchmarkSeries{BenchmarkID: id, Basis: "price", Points: filtered, Meta: SourceMeta{Provider: "aktools", SourceURL: source, ObservedAt: filtered[len(filtered)-1].Date, FetchedAt: c.Now().UTC()}}, nil
}

func first(row map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := row[k]; ok {
			return v
		}
	}
	return nil
}
func valueString(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	return s, s != "" && s != "<nil>"
}
func textField(row map[string]any, keys ...string) (string, bool) {
	return valueString(first(row, keys...))
}
func anyText(v any) string { s, _ := valueString(v); return s }
func inferShareClass(name string) string {
	upper := strings.ToUpper(strings.TrimSpace(name))
	for _, class := range []string{"A", "C", "D", "E", "F", "I"} {
		for _, suffix := range []string{class, class + "人民币", class + "(人民币)", class + "（人民币）"} {
			if strings.HasSuffix(upper, suffix) {
				return class
			}
		}
	}
	return "unknown"
}
func normalizePurchaseStatus(v string) string {
	if strings.Contains(v, "暂停") {
		return "suspended"
	}
	if strings.Contains(v, "开放") || strings.Contains(v, "限额") || strings.Contains(v, "限大额") {
		return "open"
	}
	return "unknown"
}
func parseFloatPtr(v any) *float64 {
	s, ok := valueString(v)
	if !ok || s == "--" {
		return nil
	}
	s = strings.ReplaceAll(strings.TrimSuffix(s, "%"), ",", "")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}
func parsePercent(v any) *float64 {
	p := parseFloatPtr(v)
	if p != nil {
		*p /= 100
	}
	return p
}
func parseCNYCents(v any) *int64 {
	p := parseFloatPtr(v)
	if p == nil {
		return nil
	}
	cents := int64(*p*100 + 0.5)
	return &cents
}
func parsePoints(rows []map[string]any, dateKeys, valueKeys []string) ([]SeriesPoint, error) {
	points := make([]SeriesPoint, 0, len(rows))
	for _, row := range rows {
		d, ok := textField(row, dateKeys...)
		if !ok {
			return nil, fmt.Errorf("series date contract changed")
		}
		date, err := parseDate(d)
		if err != nil {
			return nil, fmt.Errorf("series date contract changed")
		}
		v := parseFloatPtr(first(row, valueKeys...))
		if v == nil {
			return nil, fmt.Errorf("series value contract changed")
		}
		points = append(points, SeriesPoint{Date: date, Value: *v})
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("empty series")
	}
	return points, nil
}
func parseDate(v string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", "2006-01-02T15:04:05.000", time.RFC3339, "20060102", "2006/01/02"} {
		if parsed, err := time.Parse(layout, v); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}

func parseOptionalDate(v any) *time.Time {
	raw, ok := valueString(v)
	if !ok || raw == "--" {
		return nil
	}
	parsed, err := parseDate(raw)
	if err != nil {
		return nil
	}
	return &parsed
}
