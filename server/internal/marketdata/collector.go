package marketdata

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"investplanner/server/internal/database"
	"investplanner/server/internal/repository"
)

const systemActor = "00000000-0000-0000-0000-000000000000"

type Collector struct {
	Store       *repository.Store
	Snapshots   *SnapshotService
	AKTools     *AKToolsClient
	Tushare     *TushareClient
	ChinaMoney  *ChinaMoneyClient
	Interval    time.Duration
	Concurrency int
	Now         func() time.Time
}
type normalizedPurchase struct {
	Channel         string     `json:"channel"`
	Status          string     `json:"status"`
	NextOpenDate    *time.Time `json:"nextOpenDate"`
	MinimumCents    *int64     `json:"minimumCents"`
	DailyLimitCents *int64     `json:"dailyLimitCents"`
}
type normalizedETF struct {
	Price       *float64 `json:"price"`
	IOPV        *float64 `json:"iopv"`
	PremiumRate *float64 `json:"premiumRate"`
}
type normalizedFX struct {
	CNYPerUnit float64 `json:"cnyPerUnit"`
	Unit       string  `json:"unit"`
}

func (c *Collector) Refresh(ctx context.Context) []error {
	jobs := []func(context.Context) error{c.refreshFreeTracking, c.refreshPurchases, c.refreshETF}
	if c.ChinaMoney != nil {
		jobs = append(jobs, c.refreshFX)
	}
	if c.Tushare != nil {
		jobs = append(jobs, c.refreshTushareNAV, c.refreshTushareBenchmarks, c.refreshTushareCalendar, c.refreshTushareFX)
	}
	limit := c.Concurrency
	if limit < 1 {
		limit = 2
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := []error{}
	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			if err := job(ctx); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return errs
}

func (c *Collector) refreshFreeTracking(ctx context.Context) error {
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	fx, err := c.AKTools.USDCNYReferenceHistory(ctx)
	if err != nil {
		return fmt.Errorf("free tracking FX: %w", err)
	}
	benchmarks := map[string]BenchmarkSeries{}
	for benchmarkID, symbol := range map[string]string{"NASDAQ100_TR": "QQQ", "SP500_TR": "SPY"} {
		etf, fetchErr := c.AKTools.USFundAdjustedClose(ctx, symbol)
		if fetchErr != nil {
			continue
		}
		series, alignErr := CNYAdjustedBenchmark(etf, fx)
		if alignErr != nil {
			continue
		}
		series.BenchmarkID = benchmarkID
		benchmarks[benchmarkID] = series
		if err := c.Snapshots.Put("benchmark_etf_proxy_cny", benchmarkID, series, series.Meta); err != nil {
			return fmt.Errorf("free tracking benchmark snapshot: %w", err)
		}
	}
	shares, err := c.Store.ListActiveFundShares(now)
	if err != nil {
		return fmt.Errorf("free tracking fund list: %w", err)
	}
	mappings, err := c.Store.ListActiveBenchmarkMappings(now)
	if err != nil {
		return fmt.Errorf("free tracking mapping list: %w", err)
	}
	benchmarkByShare := map[string]string{}
	for _, mapping := range mappings {
		benchmarkByShare[mapping.FundShareID] = mapping.BenchmarkID
	}
	etfNAVAvailable := true
	etfAdjustedAvailable := true
	for _, share := range shares {
		benchmarkID := benchmarkByShare[share.ID]
		base, ok := benchmarks[benchmarkID]
		if !ok {
			continue
		}
		var series NAVSeries
		var fetchErr error
		var snapshotKind string
		switch share.Venue {
		case "otc":
			series, fetchErr = c.AKTools.FundCumulativeNAV(ctx, share.FundCode)
			snapshotKind = "fund_cumulative_nav_proxy"
		case "etf":
			if etfNAVAvailable {
				series, fetchErr = c.AKTools.FundETFCumulativeNAV(ctx, share.FundCode)
				if fetchErr != nil {
					etfNAVAvailable = false
				}
			}
			if series.Basis == "" && etfAdjustedAvailable {
				series, fetchErr = c.AKTools.FundETFAdjustedClose(ctx, share.FundCode)
				if fetchErr != nil {
					etfAdjustedAvailable = false
				}
			}
			if series.Basis == "" {
				series, fetchErr = c.AKTools.FundETFUnadjustedCloseNoActions(ctx, share.FundCode)
			}
			snapshotKind = "fund_etf_tracking_series"
		default:
			continue
		}
		if fetchErr != nil {
			continue
		}
		if err := c.Snapshots.Put(snapshotKind, share.FundCode, series, series.Meta); err != nil {
			return fmt.Errorf("free tracking fund snapshot: %w", err)
		}
		value, count, through, reason := AnnualizedTrackingError(series.Points, series.Basis, base.Points, base.Basis, now)
		if share.Venue == "etf" && series.Basis != "cumulative_nav_proxy" {
			value, count, through, reason = AnnualizedQDIIETFTrackingError(series.Points, series.Basis, base.Points, base.Basis, now)
		}
		if reason != "" {
			continue
		}
		basis := "fund_nav_proxy"
		if share.Venue == "etf" && series.Basis != "cumulative_nav_proxy" {
			basis = "etf_market_price_proxy"
		}
		result := TrackingErrorResult{Value: value, Count: count, Through: through, Basis: basis, BenchmarkSymbol: map[string]string{"NASDAQ100_TR": "QQQ", "SP500_TR": "SPY"}[benchmarkID]}
		meta := SourceMeta{Provider: series.Meta.Provider + "+" + base.Meta.Provider, SourceURL: series.Meta.SourceURL + " | " + base.Meta.SourceURL, ObservedAt: through, FetchedAt: now}
		if err := c.Snapshots.Put("tracking_error_proxy", share.FundCode+"|"+benchmarkID, result, meta); err != nil {
			return fmt.Errorf("free tracking result snapshot: %w", err)
		}
	}
	return nil
}
func (c *Collector) Run(ctx context.Context) {
	interval := c.Interval
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	for _, err := range c.Refresh(ctx) {
		log.Printf("market data refresh failed: %v", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, err := range c.Refresh(ctx) {
				log.Printf("market data refresh failed: %v", err)
			}
		}
	}
}
func (c *Collector) refreshPurchases(ctx context.Context) error {
	items, err := c.AKTools.PurchaseLimits(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	for _, item := range items {
		if item.Venue != "otc" || !isRMBShare(item.Name) || !possibleSupportedBenchmark(item.Name) {
			continue
		}
		overview, err := c.AKTools.Overview(ctx, item.FundCode)
		if err != nil {
			continue
		}
		benchmarkID, ok := exactBenchmarkID(overview.TrackingTarget)
		if !ok {
			continue
		}
		share := database.FundShare{FundCode: item.FundCode, ShareClass: item.ShareClass, Name: item.Name, Venue: "otc", Active: true, EffectiveFrom: now, CreatedBy: systemActor, UpdatedBy: systemActor}
		if err := c.Store.UpsertVerifiedFundShareMapping(share, benchmarkID, now); err != nil {
			return err
		}
	}
	shares, err := c.Store.ListActiveFundShares(now)
	if err != nil {
		return err
	}
	activeCodes := make(map[string]bool, len(shares))
	for _, share := range shares {
		activeCodes[share.FundCode] = true
	}
	for _, item := range items {
		if item.Venue != "otc" || !activeCodes[item.FundCode] {
			continue
		}
		value := normalizedPurchase{Channel: item.Channel, Status: item.Status, NextOpenDate: item.NextOpenDate, MinimumCents: item.MinimumCents, DailyLimitCents: item.DailyLimitCents}
		if err := c.Snapshots.Put("purchase_limit", item.FundCode, value, item.Meta); err != nil {
			return err
		}
	}
	return nil
}
func (c *Collector) refreshETF(ctx context.Context) error {
	items, err := c.AKTools.ETFQuotes(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	for _, item := range items {
		name := item.Name
		if name == "" {
			name = item.FundCode
		}
		if err := c.Store.UpsertDiscoveredFundShare(database.FundShare{FundCode: item.FundCode, ShareClass: "ETF", Name: name, Venue: "etf", Active: true, EffectiveFrom: now, CreatedBy: systemActor, UpdatedBy: systemActor}); err != nil {
			return err
		}
		if possibleSupportedBenchmark(name) {
			overview, overviewErr := c.AKTools.Overview(ctx, item.FundCode)
			if overviewErr == nil {
				if benchmarkID, ok := exactBenchmarkID(overview.TrackingTarget); ok {
					share := database.FundShare{FundCode: item.FundCode, ShareClass: "ETF", Name: name, Venue: "etf", Active: true, EffectiveFrom: now, CreatedBy: systemActor, UpdatedBy: systemActor}
					if err := c.Store.UpsertVerifiedFundShareMapping(share, benchmarkID, now); err != nil {
						return err
					}
				}
			}
		}
		if err := c.Snapshots.Put("etf_quote", item.FundCode, normalizedETF{Price: item.Price, IOPV: item.IOPV, PremiumRate: item.PremiumRate}, item.Meta); err != nil {
			return err
		}
	}
	return nil
}

func possibleSupportedBenchmark(name string) bool {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(name), " ", ""))
	return strings.Contains(normalized, "纳斯达克100") || strings.Contains(normalized, "纳指100") ||
		strings.Contains(normalized, "纳指") || strings.Contains(normalized, "标普500") ||
		strings.Contains(normalized, "标准普尔500") || strings.Contains(normalized, "S&P500")
}

func isRMBShare(name string) bool {
	return !strings.Contains(name, "美元") && !strings.Contains(name, "美钞") && !strings.Contains(name, "美汇")
}

func exactBenchmarkID(target string) (string, bool) {
	normalized := strings.NewReplacer(" ", "", "-", "", "_", "", "（", "", "）", "", "(", "", ")", "").Replace(strings.ToUpper(strings.TrimSpace(target)))
	switch normalized {
	case "纳斯达克100指数", "NASDAQ100指数", "NASDAQ100INDEX":
		return "NASDAQ100_TR", true
	case "标准普尔500指数", "标普500指数", "S&P500指数", "S&P500INDEX":
		return "SP500_TR", true
	default:
		return "", false
	}
}
func (c *Collector) refreshFX(ctx context.Context) error {
	items, err := c.ChinaMoney.ReferenceRates(ctx, time.Now())
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := c.Snapshots.Put("fx_rate", item.Currency, normalizedFX{CNYPerUnit: item.CNYPerUnit, Unit: item.Unit}, item.Meta); err != nil {
			return err
		}
	}
	return nil
}
func (c *Collector) refreshTushareNAV(ctx context.Context) error {
	now := time.Now().UTC()
	shares, err := c.Store.ListActiveFundShares(now)
	if err != nil {
		return err
	}
	for _, share := range shares {
		if share.Venue != "otc" {
			continue
		}
		series, err := c.Tushare.FundNAV(ctx, share.FundCode)
		if err != nil {
			continue
		}
		if err := c.Snapshots.Put("fund_adj_nav", share.FundCode, series, series.Meta); err != nil {
			return err
		}
	}
	return nil
}
func (c *Collector) refreshTushareBenchmarks(ctx context.Context) error {
	now := time.Now().UTC()
	mappings, err := c.Store.ListActiveBenchmarkMappings(now)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, mapping := range mappings {
		if seen[mapping.BenchmarkID] {
			continue
		}
		seen[mapping.BenchmarkID] = true
		series, err := c.Tushare.Benchmark(ctx, mapping.BenchmarkID, now.AddDate(-1, 0, 0), now)
		if err != nil {
			continue
		}
		if err := c.Snapshots.Put("benchmark_price", mapping.BenchmarkID, series, series.Meta); err != nil {
			return err
		}
	}
	return nil
}
func (c *Collector) refreshTushareCalendar(ctx context.Context) error {
	now := time.Now().UTC()
	days, err := c.Tushare.TradeCalendar(ctx, now, now.AddDate(1, 0, 0))
	if err != nil {
		return err
	}
	meta := SourceMeta{Provider: "tushare", SourceURL: "https://tushare.pro/document/2?doc_id=26", ObservedAt: now, FetchedAt: now}
	return c.Snapshots.Put("trade_calendar", "SSE", days, meta)
}
func (c *Collector) refreshTushareFX(ctx context.Context) error {
	now := time.Now().UTC()
	rates, err := c.Tushare.ReferenceRates(ctx, now)
	if err != nil {
		return err
	}
	for _, rate := range rates {
		if err := c.Snapshots.Put("fx_rate_fallback", rate.Currency, normalizedFX{CNYPerUnit: rate.CNYPerUnit, Unit: rate.Unit}, rate.Meta); err != nil {
			return err
		}
	}
	return nil
}
