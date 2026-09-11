package marketdata

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"investplanner/server/internal/database"
)

const systemActor = "00000000-0000-0000-0000-000000000000"

type Collector struct {
	Store           CollectorStore
	Snapshots       *SnapshotService
	AKTools         *AKToolsClient
	PurchaseDetails PurchaseDetailProvider
	Tushare         *TushareClient
	ChinaMoney      *ChinaMoneyClient
	Interval        time.Duration
	Concurrency     int
	Now             func() time.Time
}

type CollectorStore interface {
	ListActiveFundShares(time.Time) ([]database.FundShare, error)
	ListActiveBenchmarkMappings(time.Time) ([]database.BenchmarkMapping, error)
	UpsertVerifiedFundShareMapping(database.FundShare, string, time.Time) error
	UpsertDiscoveredFundShare(database.FundShare) error
}
type normalizedPurchase struct {
	Channel         string     `json:"channel"`
	Status          string     `json:"status"`
	RawStatus       string     `json:"rawStatus"`
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
	// Populate the two user-facing assistants before the much slower historical
	// series jobs can occupy every collector slot. This keeps first startup from
	// showing an empty QDII/FX panel for several minutes.
	errs := c.runJobs(ctx, []func(context.Context) error{c.refreshPurchases, c.refreshFX})
	// The free tracking job and the broad ETF quote job both fan out to
	// EastMoney. Running them together makes the NAV endpoint substantially
	// less reliable, so finish the smaller tracking set before starting the
	// broad quote refresh.
	errs = append(errs, c.runJobs(ctx, []func(context.Context) error{c.refreshFreeTracking})...)
	jobs := []func(context.Context) error{c.refreshETF}
	if c.Tushare != nil {
		jobs = append(jobs, c.refreshTushareNAV, c.refreshTushareBenchmarks, c.refreshTushareCalendar, c.refreshTushareFX)
	}
	return append(errs, c.runJobs(ctx, jobs)...)
}

func (c *Collector) runJobs(ctx context.Context, jobs []func(context.Context) error) []error {
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
	if len(benchmarks) == 0 {
		return fmt.Errorf("free tracking: no valid benchmark series")
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
	completed := map[string]bool{}
	for pass := 0; pass < 2; pass++ {
		if pass > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
		}
		for _, share := range shares {
			benchmarkID := benchmarkByShare[share.ID]
			base, ok := benchmarks[benchmarkID]
			if !ok {
				continue
			}
			resultKey := share.FundCode + "|" + benchmarkID
			if completed[resultKey] {
				continue
			}
			series, snapshotKind, fetchErr := c.freeTrackingSeries(ctx, share)
			if fetchErr != nil {
				if pass == 1 {
					log.Printf("free tracking fund %s (%s) unavailable: %v", share.FundCode, share.Venue, fetchErr)
				}
				continue
			}
			if err := c.Snapshots.Put(snapshotKind, share.FundCode, series, series.Meta); err != nil {
				return fmt.Errorf("free tracking fund snapshot: %w", err)
			}
			value, count, through, reason := AnnualizedTrackingError(series.Points, series.Basis, base.Points, base.Basis, now)
			if reason != "" {
				if pass == 1 {
					log.Printf("free tracking fund %s (%s) skipped: %s", share.FundCode, share.Venue, reason)
				}
				continue
			}
			basis := "fund_nav_proxy"
			result := TrackingErrorResult{MethodVersion: TrackingMethodVersion, Value: value, Count: count, Through: through, Basis: basis, BenchmarkSymbol: map[string]string{"NASDAQ100_TR": "QQQ", "SP500_TR": "SPY"}[benchmarkID]}
			meta := SourceMeta{Provider: series.Meta.Provider + "+" + base.Meta.Provider, SourceURL: series.Meta.SourceURL + " | " + base.Meta.SourceURL, ObservedAt: through, FetchedAt: now}
			if err := c.Snapshots.Put("tracking_error_proxy", resultKey, result, meta); err != nil {
				return fmt.Errorf("free tracking result snapshot: %w", err)
			}
			completed[resultKey] = true
		}
	}
	return nil
}

func (c *Collector) freeTrackingSeries(ctx context.Context, share database.FundShare) (NAVSeries, string, error) {
	switch share.Venue {
	case "otc":
		series, err := c.AKTools.FundCumulativeNAV(ctx, share.FundCode)
		return series, "fund_cumulative_nav_proxy", err
	case "etf":
		// The public cumulative-NAV route covers exchange-traded QDII funds more
		// reliably than fund_etf_fund_info_em in the pinned AKShare release. Keep
		// the ETF-specific route only as a NAV fallback. Market-price deviation is
		// deliberately not returned as fund tracking error.
		series, err := c.AKTools.FundCumulativeNAV(ctx, share.FundCode)
		if err != nil {
			series, err = c.AKTools.FundETFCumulativeNAV(ctx, share.FundCode)
		}
		return series, "fund_etf_tracking_series", err
	default:
		return NAVSeries{}, "", fmt.Errorf("unsupported fund venue")
	}
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
	catalog, err := c.AKTools.DiscoverQDII(ctx)
	if err != nil {
		return err
	}
	catalogByCode := make(map[string]FundCatalogItem, len(catalog))
	for _, item := range catalog {
		if isRMBShare(item.Name) {
			catalogByCode[item.Code] = item
		}
	}
	items, err := c.AKTools.PurchaseLimits(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	shares, err := c.Store.ListActiveFundShares(now)
	if err != nil {
		return err
	}
	activeCodes := make(map[string]bool, len(shares))
	for _, share := range shares {
		if share.Venue == "otc" {
			activeCodes[share.FundCode] = true
		}
	}
	snapshottedCodes := make(map[string]bool, len(activeCodes))
	// Refresh already verified funds first. Discovering brand-new mappings can
	// require many sequential overview requests and must not hold existing user
	// rows on an old purchase snapshot.
	for _, item := range items {
		if item.Venue != "otc" || !activeCodes[item.FundCode] || snapshottedCodes[item.FundCode] {
			continue
		}
		if err := c.putPurchase(item); err != nil {
			return err
		}
		snapshottedCodes[item.FundCode] = true
	}
	if err := c.refreshPurchaseDetails(ctx, items, snapshottedCodes); err != nil {
		return err
	}
	for _, item := range items {
		catalogItem, inCatalog := catalogByCode[item.FundCode]
		if item.Venue != "otc" || !inCatalog || snapshottedCodes[item.FundCode] {
			continue
		}
		candidateName := catalogItem.Name
		if candidateName == "" {
			candidateName = item.Name
		}
		if !possibleSupportedBenchmark(candidateName) {
			continue
		}
		overview, overviewErr := c.AKTools.Overview(ctx, item.FundCode)
		if overviewErr != nil {
			continue
		}
		benchmarkID, ok := exactBenchmarkID(overview.TrackingTarget)
		if !ok {
			continue
		}
		shareClass := catalogItem.ShareClass
		if shareClass == "" || shareClass == "unknown" {
			shareClass = item.ShareClass
		}
		share := database.FundShare{FundCode: item.FundCode, ShareClass: shareClass, Name: candidateName, Venue: "otc", Active: true, EffectiveFrom: now, CreatedBy: systemActor, UpdatedBy: systemActor}
		if err := c.Store.UpsertVerifiedFundShareMapping(share, benchmarkID, now); err != nil {
			return err
		}
		// Do not re-query by the just-persisted effective timestamp. MySQL's
		// millisecond rounding can otherwise exclude this mapping until the
		// next six-hour refresh.
		activeCodes[item.FundCode] = true
		if err := c.putPurchase(item); err != nil {
			return err
		}
		snapshottedCodes[item.FundCode] = true
	}
	return nil
}

func (c *Collector) refreshPurchaseDetails(ctx context.Context, items []PurchaseLimit, mapped map[string]bool) error {
	if c.PurchaseDetails == nil {
		return nil
	}
	// A failing detail site must not hold up the other data jobs indefinitely.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, item := range items {
		if ctx.Err() != nil {
			break
		}
		if !mapped[item.FundCode] || item.DailyLimitCents != nil {
			continue
		}
		detail, err := c.PurchaseDetails.PurchaseDetail(ctx, item.FundCode)
		if err != nil {
			continue
		}
		if err := c.putPurchase(detail); err != nil {
			return err
		}
	}
	return nil
}

func (c *Collector) putPurchase(item PurchaseLimit) error {
	value := normalizedPurchase{Channel: item.Channel, Status: item.Status, RawStatus: item.RawStatus, NextOpenDate: item.NextOpenDate, MinimumCents: item.MinimumCents, DailyLimitCents: item.DailyLimitCents}
	return c.Snapshots.Put("purchase_limit", item.FundCode, value, item.Meta)
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
	var items []FXRate
	var err error
	if c.ChinaMoney != nil {
		items, err = c.ChinaMoney.ReferenceRates(ctx, time.Now())
	}
	if len(items) == 0 {
		if c.AKTools == nil {
			if err != nil {
				return err
			}
			return fmt.Errorf("FX provider unavailable")
		}
		items, err = c.AKTools.SAFEReferenceRates(ctx)
	}
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
