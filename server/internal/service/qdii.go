package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"investplanner/server/internal/database"
	"investplanner/server/internal/domain"
	"investplanner/server/internal/marketdata"
	"investplanner/server/internal/repository"
)

type QDIIService struct {
	Store    *repository.Store
	Now      func() time.Time
	FreshFor time.Duration
}
type QDIICandidate struct {
	FundCode                string                  `json:"fundCode"`
	Name                    string                  `json:"name"`
	ShareClass              string                  `json:"shareClass"`
	Venue                   string                  `json:"venue"`
	Channel                 string                  `json:"channel,omitempty"`
	PurchaseStatus          string                  `json:"purchaseStatus,omitempty"`
	NextOpenDate            AssistantValue[string]  `json:"nextOpenDate"`
	DailyLimitCents         AssistantValue[int64]   `json:"dailyLimitCents"`
	MinimumCents            AssistantValue[int64]   `json:"minimumCents"`
	RemainingNeedCents      int64                   `json:"remainingNeedCents"`
	RemainingPurchaseDays   AssistantValue[int]     `json:"remainingPurchaseDays"`
	ExecutableCapacityCents AssistantValue[int64]   `json:"executableCapacityCents"`
	TrackingError           AssistantValue[float64] `json:"trackingError"`
	Price                   AssistantValue[float64] `json:"price"`
	IOPV                    AssistantValue[float64] `json:"iopv"`
	PremiumRate             AssistantValue[float64] `json:"premiumRate"`
}
type QDIIResult struct {
	DestinationID string          `json:"destinationId"`
	BenchmarkID   string          `json:"benchmarkId"`
	Status        string          `json:"status"`
	Reason        string          `json:"reason,omitempty"`
	OTC           []QDIICandidate `json:"otc"`
	ETFs          []QDIICandidate `json:"etfs"`
}
type purchaseSnapshot struct {
	Channel         string     `json:"channel"`
	Status          string     `json:"status"`
	NextOpenDate    *time.Time `json:"nextOpenDate"`
	MinimumCents    *int64     `json:"minimumCents"`
	DailyLimitCents *int64     `json:"dailyLimitCents"`
}
type etfSnapshot struct{ Price, IOPV, PremiumRate *float64 }
type seriesSnapshot struct {
	Basis  string                   `json:"basis"`
	Points []marketdata.SeriesPoint `json:"points"`
}

func (s *QDIIService) Assist(userID, planID, destinationID string, remainingNeed int64) (QDIIResult, error) {
	plan, err := s.Store.GetPlan(userID, planID)
	if err != nil {
		return QDIIResult{}, err
	}
	var destination *database.PlanDestination
	for i := range plan.Destinations {
		if plan.Destinations[i].ID == destinationID {
			destination = &plan.Destinations[i]
			break
		}
	}
	if destination == nil {
		return QDIIResult{}, repository.ErrNotFound
	}
	result := QDIIResult{DestinationID: destinationID, BenchmarkID: destination.BenchmarkID, OTC: []QDIICandidate{}, ETFs: []QDIICandidate{}}
	if destination.ExecutionMode != domain.ExecutionQDII {
		return result, fmt.Errorf("%w: 标的不是 QDII", ErrValidation)
	}
	if destination.BenchmarkID == "" {
		result.Status = "unavailable"
		result.Reason = "未配置精确基准映射"
		return result, nil
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	shares, err := s.Store.ActiveFundSharesByBenchmark(destination.BenchmarkID, now)
	if err != nil {
		return result, err
	}
	for _, share := range shares {
		candidate := QDIICandidate{FundCode: share.FundCode, Name: share.Name, ShareClass: share.ShareClass, Venue: share.Venue, RemainingNeedCents: remainingNeed, NextOpenDate: unknownValue[string]("下一开放日未知"), DailyLimitCents: unknownValue[int64]("渠道限额未知"), MinimumCents: unknownValue[int64]("最低申购额未知"), RemainingPurchaseDays: unknownValue[int]("剩余可申购日未知"), ExecutableCapacityCents: unknownValue[int64]("无法估算容量"), TrackingError: s.trackingError(share.FundCode, destination.BenchmarkID), Price: unknownValue[float64]("行情未知"), IOPV: unknownValue[float64]("IOPV 未知"), PremiumRate: unknownValue[float64]("溢折率未知")}
		if share.Venue == "etf" {
			s.addETF(&candidate, share.FundCode)
		} else {
			s.addPurchase(&candidate, share.FundCode, now)
		}
		if share.Venue == "etf" {
			result.ETFs = append(result.ETFs, candidate)
		} else {
			result.OTC = append(result.OTC, candidate)
		}
	}
	sort.SliceStable(result.OTC, func(i, j int) bool { return betterOTC(result.OTC[i], result.OTC[j]) })
	sort.SliceStable(result.ETFs, func(i, j int) bool { return betterETF(result.ETFs[i], result.ETFs[j]) })
	if len(result.OTC) == 0 && len(result.ETFs) == 0 {
		result.Status = "unavailable"
		result.Reason = "没有已映射且可用的同指数基金快照"
	} else {
		result.Status = "fresh"
	}
	return result, nil
}
func (s *QDIIService) snapshot(dataType, key string, destination any) (database.MarketDataSnapshot, error) {
	snapshot, err := s.Store.GetSnapshot(dataType, key)
	if err != nil {
		return snapshot, err
	}
	if json.Unmarshal(snapshot.NormalizedJSON, destination) != nil {
		return snapshot, fmt.Errorf("invalid snapshot")
	}
	return snapshot, nil
}
func (s *QDIIService) freshness(snapshot database.MarketDataSnapshot) string {
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	freshFor := s.FreshFor
	if freshFor <= 0 {
		freshFor = 24 * time.Hour
	}
	return string(marketdata.Classify(snapshot.FetchedAt, now, freshFor))
}
func (s *QDIIService) addPurchase(c *QDIICandidate, code string, now time.Time) {
	var data purchaseSnapshot
	snap, err := s.snapshot("purchase_limit", code, &data)
	if err != nil {
		return
	}
	fresh := s.freshness(snap)
	c.Channel = data.Channel
	c.PurchaseStatus = data.Status
	if data.NextOpenDate != nil {
		v := data.NextOpenDate.Format("2006-01-02")
		c.NextOpenDate = knownValue(v, v, snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if data.DailyLimitCents != nil {
		c.DailyLimitCents = knownValue(*data.DailyLimitCents, fmt.Sprintf("%d", *data.DailyLimitCents), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if data.MinimumCents != nil {
		c.MinimumCents = knownValue(*data.MinimumCents, fmt.Sprintf("%d", *data.MinimumCents), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if data.Status == "suspended" {
		c.RemainingPurchaseDays = knownValue(0, "0", snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
		zero := int64(0)
		c.ExecutableCapacityCents = knownValue(zero, "0", snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
		return
	}
	if data.Status == "open" && data.DailyLimitCents != nil {
		days := remainingWeekdays(now)
		c.RemainingPurchaseDays = knownValue(days, fmt.Sprintf("%d", days), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
		capacity := *data.DailyLimitCents * int64(days)
		if capacity > c.RemainingNeedCents {
			capacity = c.RemainingNeedCents
		}
		c.ExecutableCapacityCents = knownValue(capacity, fmt.Sprintf("%d", capacity), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
}
func (s *QDIIService) addETF(c *QDIICandidate, code string) {
	var data etfSnapshot
	snap, err := s.snapshot("etf_quote", code, &data)
	if err != nil {
		return
	}
	fresh := s.freshness(snap)
	if data.Price != nil {
		c.Price = knownValue(*data.Price, fmt.Sprintf("%.4f", *data.Price), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if data.IOPV != nil {
		c.IOPV = knownValue(*data.IOPV, fmt.Sprintf("%.4f", *data.IOPV), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if data.PremiumRate != nil && data.IOPV != nil {
		c.PremiumRate = knownValue(*data.PremiumRate, fmt.Sprintf("%.2f%%", *data.PremiumRate*100), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
}
func (s *QDIIService) trackingError(code, benchmark string) AssistantValue[float64] {
	var fund, base seriesSnapshot
	fundSnap, err := s.snapshot("fund_adj_nav", code, &fund)
	baseSnap, baseErr := s.snapshot("benchmark_total_return", benchmark, &base)
	proxy := false
	if err != nil || baseErr != nil {
		var cached marketdata.TrackingErrorResult
		cachedSnap, cachedErr := s.snapshot("tracking_error_proxy", code+"|"+benchmark, &cached)
		if cachedErr == nil {
			fresh := s.freshness(cachedSnap)
			basisLabel := ""
			if cached.Basis == "etf_market_price_proxy" {
				basisLabel = " · 场内价格口径"
			}
			display := fmt.Sprintf("%.2f%% · 相对 %s%s · %d 个共同观测日 · %s", cached.Value*100, cached.BenchmarkSymbol, basisLabel, cached.Count, cached.Through.Format("2006-01-02"))
			return knownValue(cached.Value, display, cachedSnap.Provider, cachedSnap.SourceURL, cached.Through, fresh)
		}
		fundSnap, err = s.snapshot("fund_cumulative_nav_proxy", code, &fund)
		baseSnap, baseErr = s.snapshot("benchmark_etf_proxy_cny", benchmark, &base)
		proxy = true
	}
	if err != nil || baseErr != nil {
		return unknownValue[float64]("QQQ/SPY 免费代理收益序列暂不可用")
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	value, count, through, reason := TrackingError(fund.Points, fund.Basis, base.Points, base.Basis, now)
	if reason != "" {
		return unknownValue[float64](reason)
	}
	fresh := s.freshness(fundSnap)
	if s.freshness(baseSnap) == "stale" {
		fresh = "stale"
	}
	display := fmt.Sprintf("%.2f%% · %d 个共同观测日 · %s", value*100, count, through.Format("2006-01-02"))
	if proxy {
		label := map[string]string{"NASDAQ100_TR": "QQQ", "SP500_TR": "SPY"}[benchmark]
		display = fmt.Sprintf("%.2f%% · 相对 %s · 免费代理口径 · %d 个共同观测日 · %s", value*100, label, count, through.Format("2006-01-02"))
	}
	return knownValue(value, display, fundSnap.Provider+" + "+baseSnap.Provider, fundSnap.SourceURL+" | "+baseSnap.SourceURL, through, fresh)
}
func TrackingError(fund []marketdata.SeriesPoint, fundBasis string, benchmark []marketdata.SeriesPoint, benchmarkBasis string, now time.Time) (float64, int, time.Time, string) {
	return marketdata.AnnualizedTrackingError(fund, fundBasis, benchmark, benchmarkBasis, now)
}
func remainingWeekdays(now time.Time) int {
	last := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
	count := 0
	for day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()); !day.After(last); day = day.AddDate(0, 0, 1) {
		if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
			count++
		}
	}
	return count
}
func betterOTC(a, b QDIICandidate) bool {
	rank := func(c QDIICandidate) int {
		if c.PurchaseStatus == "open" {
			return 0
		}
		if c.PurchaseStatus == "unknown" || c.PurchaseStatus == "" {
			return 1
		}
		return 2
	}
	if rank(a) != rank(b) {
		return rank(a) < rank(b)
	}
	if (a.DailyLimitCents.Value != nil) != (b.DailyLimitCents.Value != nil) {
		return a.DailyLimitCents.Value != nil
	}
	if (a.TrackingError.Value != nil) != (b.TrackingError.Value != nil) {
		return a.TrackingError.Value != nil
	}
	if a.TrackingError.Value != nil && b.TrackingError.Value != nil && *a.TrackingError.Value != *b.TrackingError.Value {
		return *a.TrackingError.Value < *b.TrackingError.Value
	}
	return a.FundCode < b.FundCode
}
func betterETF(a, b QDIICandidate) bool {
	if a.PremiumRate.Value == nil {
		return false
	}
	if b.PremiumRate.Value == nil {
		return true
	}
	return math.Abs(*a.PremiumRate.Value) < math.Abs(*b.PremiumRate.Value)
}
