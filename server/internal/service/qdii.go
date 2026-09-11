package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
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
	PurchaseStatusInfo      AssistantValue[string]  `json:"purchaseStatusInfo"`
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
	DestinationID string            `json:"destinationId"`
	BenchmarkID   string            `json:"benchmarkId"`
	Status        string            `json:"status"`
	Reason        string            `json:"reason,omitempty"`
	OTC           []QDIICandidate   `json:"otc"`
	ETFs          []QDIICandidate   `json:"etfs"`
	Confirmation  *QDIIConfirmation `json:"confirmation,omitempty"`
}
type QDIIConfirmation struct {
	SelectedOTCFundCodes   []string               `json:"selectedOtcFundCodes"`
	SelectedOTCPlans       []QDIIConfirmedOTCPlan `json:"selectedOtcPlans"`
	SelectedETFFundCode    string                 `json:"selectedEtfFundCode"`
	SelectedETFAmountCents *int64                 `json:"selectedEtfAmountCents"`
	ConfirmedAt            time.Time              `json:"confirmedAt"`
}
type QDIIConfirmedOTCPlan struct {
	FundCode         string `json:"fundCode"`
	DailyAmountCents *int64 `json:"dailyAmountCents"`
	TotalAmountCents *int64 `json:"totalAmountCents"`
}
type QDIIConfirmedFund struct {
	FundCode         string `json:"fundCode"`
	Name             string `json:"name"`
	ShareClass       string `json:"shareClass,omitempty"`
	DailyAmountCents *int64 `json:"dailyAmountCents"`
	TotalAmountCents *int64 `json:"totalAmountCents"`
}
type QDIIConfirmationSummary struct {
	DestinationID          string              `json:"destinationId"`
	SelectedOTCFunds       []QDIIConfirmedFund `json:"selectedOtcFunds"`
	SelectedETFFund        *QDIIConfirmedFund  `json:"selectedEtfFund,omitempty"`
	SelectedETFAmountCents *int64              `json:"selectedEtfAmountCents"`
	ConfirmedAt            time.Time           `json:"confirmedAt"`
}
type purchaseSnapshot struct {
	Channel         string     `json:"channel"`
	Status          string     `json:"status"`
	RawStatus       string     `json:"rawStatus"`
	NextOpenDate    *time.Time `json:"nextOpenDate"`
	MinimumCents    *int64     `json:"minimumCents"`
	DailyLimitCents *int64     `json:"dailyLimitCents"`
}
type etfSnapshot struct{ Price, IOPV, PremiumRate *float64 }
type seriesSnapshot struct {
	Basis  string                   `json:"basis"`
	Points []marketdata.SeriesPoint `json:"points"`
}

func (s *QDIIService) Assist(userID, planID, destinationID string, remainingNeed int64, month ...string) (QDIIResult, error) {
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
	now = now.In(time.FixedZone("Asia/Shanghai", 8*60*60))
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
			if len(month) > 0 && month[0] != "" && month[0] != now.Format("2006-01") {
				candidate.RemainingPurchaseDays = unknownValue[int]("当前申购快照不能推算其他月份")
				candidate.ExecutableCapacityCents = unknownValue[int64]("当前申购快照不能推算其他月份")
			}
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
		for _, c := range append(append([]QDIICandidate(nil), result.OTC...), result.ETFs...) {
			if c.TrackingError.Freshness != "fresh" || (c.Venue == "otc" && c.PurchaseStatusInfo.Freshness != "fresh") || (c.Venue == "etf" && c.Price.Freshness != "fresh") {
				result.Status = "stale"
				result.Reason = "部分数据缺失或陈旧，请查看各项状态与观测日期"
			}
		}
	}
	if len(month) > 0 && month[0] != "" {
		confirmation, err := s.Store.GetQDIIConfirmation(userID, planID, month[0], destinationID)
		if err == nil {
			var selectedOTC []string
			if json.Unmarshal(confirmation.SelectedOTCFundCodesJSON, &selectedOTC) == nil {
				selectedOTCPlans, parseErr := parseSelectedOTCPlans(confirmation.SelectedOTCPlansJSON)
				if parseErr != nil {
					return result, parseErr
				}
				result.Confirmation = &QDIIConfirmation{SelectedOTCFundCodes: selectedOTC, SelectedOTCPlans: selectedOTCPlans, SelectedETFFundCode: confirmation.SelectedETFFundCode, SelectedETFAmountCents: confirmation.SelectedETFAmountCents, ConfirmedAt: confirmation.ConfirmedAt}
			}
		} else if !errors.Is(err, repository.ErrNotFound) {
			return result, err
		}
	}
	return result, nil
}

func (s *QDIIService) Confirm(userID, planID, destinationID, month string, selectedOTC []string, selectedETF string) (QDIIConfirmation, error) {
	if _, err := time.Parse("2006-01", month); err != nil {
		return QDIIConfirmation{}, fmt.Errorf("%w: 计划月份无效", ErrValidation)
	}
	record, err := s.Store.GetMonth(userID, planID, month)
	if err != nil {
		return QDIIConfirmation{}, err
	}
	remainingNeed := int64(0)
	allocationFound := false
	for _, allocation := range record.Allocations {
		if allocation.DestinationID != nil && *allocation.DestinationID == destinationID {
			remainingNeed = max(allocation.RecommendedCents-allocation.ActualCents, 0)
			allocationFound = true
			break
		}
	}
	if !allocationFound {
		return QDIIConfirmation{}, repository.ErrNotFound
	}
	result, err := s.Assist(userID, planID, destinationID, remainingNeed, month)
	if err != nil {
		return QDIIConfirmation{}, err
	}
	cleanOTC, selectedETF, err := validateQDIISelection(result, selectedOTC, selectedETF)
	if err != nil {
		return QDIIConfirmation{}, err
	}
	selectedOTCJSON, err := json.Marshal(cleanOTC)
	if err != nil {
		return QDIIConfirmation{}, err
	}
	selectedOTCPlans := buildSelectedOTCPlans(result, cleanOTC)
	selectedOTCPlansJSON, err := json.Marshal(selectedOTCPlans)
	if err != nil {
		return QDIIConfirmation{}, err
	}
	selectedETFAmountCents := calculateSelectedETFAmount(result, cleanOTC, selectedETF, remainingNeed)
	persisted, err := s.Store.SaveQDIIConfirmation(userID, planID, month, destinationID, selectedOTCJSON, selectedOTCPlansJSON, selectedETF, selectedETFAmountCents)
	if err != nil {
		return QDIIConfirmation{}, err
	}
	return QDIIConfirmation{SelectedOTCFundCodes: cleanOTC, SelectedOTCPlans: selectedOTCPlans, SelectedETFFundCode: selectedETF, SelectedETFAmountCents: persisted.SelectedETFAmountCents, ConfirmedAt: persisted.ConfirmedAt}, nil
}

func buildSelectedOTCPlans(result QDIIResult, selectedOTC []string) []QDIIConfirmedOTCPlan {
	candidateByCode := make(map[string]QDIICandidate, len(result.OTC))
	for _, candidate := range result.OTC {
		candidateByCode[candidate.FundCode] = candidate
	}
	plans := make([]QDIIConfirmedOTCPlan, 0, len(selectedOTC))
	for _, code := range selectedOTC {
		candidate := candidateByCode[code]
		plans = append(plans, QDIIConfirmedOTCPlan{
			FundCode:         code,
			DailyAmountCents: cloneInt64(candidate.DailyLimitCents.Value),
			TotalAmountCents: cloneInt64(candidate.ExecutableCapacityCents.Value),
		})
	}
	return plans
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func parseSelectedOTCPlans(raw []byte) ([]QDIIConfirmedOTCPlan, error) {
	if len(raw) == 0 {
		return []QDIIConfirmedOTCPlan{}, nil
	}
	var plans []QDIIConfirmedOTCPlan
	if err := json.Unmarshal(raw, &plans); err != nil {
		return nil, fmt.Errorf("invalid QDII confirmation amounts: %w", err)
	}
	if plans == nil {
		return []QDIIConfirmedOTCPlan{}, nil
	}
	return plans, nil
}

func calculateSelectedETFAmount(result QDIIResult, selectedOTC []string, selectedETF string, remainingNeed int64) *int64 {
	if selectedETF == "" {
		return nil
	}
	selected := make(map[string]bool, len(selectedOTC))
	for _, code := range selectedOTC {
		selected[code] = true
	}
	amount := max(remainingNeed, 0)
	for _, candidate := range result.OTC {
		if !selected[candidate.FundCode] {
			continue
		}
		if candidate.ExecutableCapacityCents.Value == nil || *candidate.ExecutableCapacityCents.Value < 0 {
			return nil
		}
		if *candidate.ExecutableCapacityCents.Value >= amount {
			amount = 0
		} else {
			amount -= *candidate.ExecutableCapacityCents.Value
		}
	}
	return &amount
}

func (s *QDIIService) ConfirmationSummaries(userID, planID, month string) ([]QDIIConfirmationSummary, error) {
	if _, err := time.Parse("2006-01", month); err != nil {
		return nil, fmt.Errorf("%w: 计划月份无效", ErrValidation)
	}
	confirmations, err := s.Store.ListQDIIConfirmations(userID, planID, month)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0)
	for _, confirmation := range confirmations {
		var selectedOTC []string
		if err := json.Unmarshal(confirmation.SelectedOTCFundCodesJSON, &selectedOTC); err != nil {
			return nil, fmt.Errorf("invalid QDII confirmation: %w", err)
		}
		codes = append(codes, selectedOTC...)
		if confirmation.SelectedETFFundCode != "" {
			codes = append(codes, confirmation.SelectedETFFundCode)
		}
	}
	shares, err := s.Store.FundSharesByCodes(codes)
	if err != nil {
		return nil, err
	}
	return buildQDIIConfirmationSummaries(confirmations, shares)
}

func buildQDIIConfirmationSummaries(confirmations []database.QDIIAssistantConfirmation, shares []database.FundShare) ([]QDIIConfirmationSummary, error) {
	shareByCode := make(map[string]database.FundShare, len(shares))
	for _, share := range shares {
		if _, exists := shareByCode[share.FundCode]; !exists {
			shareByCode[share.FundCode] = share
		}
	}
	fund := func(code string, plan *QDIIConfirmedOTCPlan) QDIIConfirmedFund {
		share := shareByCode[code]
		confirmed := QDIIConfirmedFund{FundCode: code, Name: share.Name, ShareClass: share.ShareClass}
		if plan != nil {
			confirmed.DailyAmountCents = cloneInt64(plan.DailyAmountCents)
			confirmed.TotalAmountCents = cloneInt64(plan.TotalAmountCents)
		}
		return confirmed
	}
	summaries := make([]QDIIConfirmationSummary, 0, len(confirmations))
	for _, confirmation := range confirmations {
		var selectedOTC []string
		if err := json.Unmarshal(confirmation.SelectedOTCFundCodesJSON, &selectedOTC); err != nil {
			return nil, fmt.Errorf("invalid QDII confirmation: %w", err)
		}
		selectedOTCPlans, err := parseSelectedOTCPlans(confirmation.SelectedOTCPlansJSON)
		if err != nil {
			return nil, err
		}
		planByCode := make(map[string]QDIIConfirmedOTCPlan, len(selectedOTCPlans))
		for _, plan := range selectedOTCPlans {
			planByCode[plan.FundCode] = plan
		}
		summary := QDIIConfirmationSummary{
			DestinationID:          confirmation.DestinationID,
			ConfirmedAt:            confirmation.ConfirmedAt,
			SelectedOTCFunds:       make([]QDIIConfirmedFund, 0, len(selectedOTC)),
			SelectedETFAmountCents: confirmation.SelectedETFAmountCents,
		}
		for _, code := range selectedOTC {
			plan, exists := planByCode[code]
			if exists {
				summary.SelectedOTCFunds = append(summary.SelectedOTCFunds, fund(code, &plan))
			} else {
				summary.SelectedOTCFunds = append(summary.SelectedOTCFunds, fund(code, nil))
			}
		}
		if confirmation.SelectedETFFundCode != "" {
			selectedETF := fund(confirmation.SelectedETFFundCode, nil)
			summary.SelectedETFFund = &selectedETF
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func validateQDIISelection(result QDIIResult, selectedOTC []string, selectedETF string) ([]string, string, error) {
	otcByCode := make(map[string]QDIICandidate, len(result.OTC))
	for _, candidate := range result.OTC {
		otcByCode[candidate.FundCode] = candidate
	}
	etfCodes := make(map[string]bool, len(result.ETFs))
	for _, candidate := range result.ETFs {
		etfCodes[candidate.FundCode] = true
	}
	cleanOTC := make([]string, 0, len(selectedOTC))
	seen := map[string]bool{}
	for _, raw := range selectedOTC {
		code := strings.TrimSpace(raw)
		candidate, ok := otcByCode[code]
		if !ok {
			return nil, "", fmt.Errorf("%w: 场外基金不属于当前精确基准映射", ErrValidation)
		}
		if candidate.PurchaseStatus == "suspended" || candidate.PurchaseStatus == "not_sold" {
			return nil, "", fmt.Errorf("%w: 暂停申购或当前渠道未代销的基金不能确认", ErrValidation)
		}
		if !seen[code] {
			cleanOTC = append(cleanOTC, code)
			seen[code] = true
		}
	}
	selectedETF = strings.TrimSpace(selectedETF)
	if selectedETF != "" && !etfCodes[selectedETF] {
		return nil, "", fmt.Errorf("%w: 场内 ETF 不属于当前精确基准映射", ErrValidation)
	}
	if len(cleanOTC) == 0 && selectedETF == "" {
		return nil, "", fmt.Errorf("%w: 请至少选择一只基金", ErrValidation)
	}
	return cleanOTC, selectedETF, nil
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
	applyPurchaseSnapshot(c, data, snap, now, s.freshness(snap))
}

// Validate again on read so legacy snapshots cannot bypass the collector's checks.
func applyPurchaseSnapshot(c *QDIICandidate, data purchaseSnapshot, snap database.MarketDataSnapshot, now time.Time, fresh string) {
	c.Channel = data.Channel
	c.PurchaseStatus = "unknown"
	c.PurchaseStatusInfo = unknownValue[string]("缺少可核验的原始申购状态，请等待重新采集")
	c.DailyLimitCents = unknownValue[int64]("源站限额缺失、为零或低于起购额，不能作为可投额度")
	c.RemainingPurchaseDays = unknownValue[int]("缺少未来开放日安排")
	c.ExecutableCapacityCents = unknownValue[int64]("申购状态或有效限额不足，无法估算月度容量")
	if data.RawStatus != "" && fresh == "fresh" {
		c.PurchaseStatus = data.Status
		c.PurchaseStatusInfo = knownValue(data.RawStatus, data.RawStatus, snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	} else if fresh != "fresh" {
		c.PurchaseStatusInfo = unknownValue[string]("申购状态快照已过期，不能视为当前状态")
	}
	if data.NextOpenDate != nil {
		v := data.NextOpenDate.Format("2006-01-02")
		c.NextOpenDate = knownValue(v, v, snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if data.DailyLimitCents != nil && *data.DailyLimitCents > 0 && (data.MinimumCents == nil || *data.DailyLimitCents >= *data.MinimumCents) {
		c.DailyLimitCents = knownValue(*data.DailyLimitCents, fmt.Sprintf("%d", *data.DailyLimitCents), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if data.MinimumCents != nil && *data.MinimumCents >= 0 {
		c.MinimumCents = knownValue(*data.MinimumCents, fmt.Sprintf("%d", *data.MinimumCents), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if c.PurchaseStatus == "suspended" {
		c.ExecutableCapacityCents = unknownValue[int64]("当前暂停申购不代表本月剩余日期均暂停；恢复安排未知")
		return
	}
	if c.PurchaseStatus == "not_sold" {
		c.MinimumCents = unknownValue[int64]("该渠道未开通代销")
		c.DailyLimitCents = unknownValue[int64]("该渠道未开通代销，非零额度")
		c.ExecutableCapacityCents = unknownValue[int64]("该渠道未开通代销，不能按此渠道估算")
		return
	}
	if c.PurchaseStatus == "open" || c.PurchaseStatus == "limited" {
		days := remainingWeekdays(now)
		c.RemainingPurchaseDays = knownValue(days, fmt.Sprintf("%d", days), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
		c.RemainingPurchaseDays.Reason = "仅统计周一至周五，未扣除节假日、基金临时暂停及当日截止时间，不是确定可申购日"
		if c.DailyLimitCents.Value == nil {
			return
		}
		capacity := max(int64(0), c.RemainingNeedCents)
		// Cap before multiplying to avoid overflow from an invalid upstream amount.
		if days == 0 {
			capacity = 0
		} else if *c.DailyLimitCents.Value <= capacity/int64(days) {
			capacity = *c.DailyLimitCents.Value * int64(days)
		}
		c.ExecutableCapacityCents = knownValue(capacity, fmt.Sprintf("%d", capacity), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
		c.ExecutableCapacityCents.Reason = "假设当前限额不变且剩余工作日均开放的估算，不是确定可投金额"
	}
}
func (s *QDIIService) addETF(c *QDIICandidate, code string) {
	var data etfSnapshot
	snap, err := s.snapshot("etf_quote", code, &data)
	if err != nil {
		return
	}
	fresh := s.freshness(snap)
	if validPositive(data.Price) {
		c.Price = knownValue(*data.Price, fmt.Sprintf("%.4f", *data.Price), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if validPositive(data.IOPV) {
		c.IOPV = knownValue(*data.IOPV, fmt.Sprintf("%.4f", *data.IOPV), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
	}
	if fresh == "fresh" && validPositive(data.Price) && validPositive(data.IOPV) {
		premium := *data.Price / *data.IOPV - 1
		if !math.IsInf(premium, 0) {
			c.PremiumRate = knownValue(premium, fmt.Sprintf("%.2f%%", premium*100), snap.Provider, snap.SourceURL, snap.ObservedAt, fresh)
		}
	}
}
func validPositive(value *float64) bool {
	return value != nil && *value > 0 && !math.IsInf(*value, 0) && !math.IsNaN(*value)
}
func (s *QDIIService) trackingError(code, benchmark string) AssistantValue[float64] {
	var fund, base seriesSnapshot
	fundSnap, err := s.snapshot("fund_adj_nav", code, &fund)
	baseSnap, baseErr := s.snapshot("benchmark_total_return", benchmark, &base)
	proxy := false
	if err != nil || baseErr != nil {
		var cached marketdata.TrackingErrorResult
		cachedSnap, cachedErr := s.snapshot("tracking_error_proxy", code+"|"+benchmark, &cached)
		now := time.Now()
		if s.Now != nil {
			now = s.Now()
		}
		if cachedErr == nil && validCachedTracking(cached, benchmark, now) {
			fresh := s.freshness(cachedSnap)
			basisLabel := " · 累计净值代理口径"
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
func validCachedTracking(value marketdata.TrackingErrorResult, benchmark string, now time.Time) bool {
	return value.MethodVersion == marketdata.TrackingMethodVersion && value.Count >= 120 && value.Value >= 0 && !math.IsNaN(value.Value) && !math.IsInf(value.Value, 0) && !value.Through.After(now) && now.Sub(value.Through) <= 14*24*time.Hour && value.Basis == "fund_nav_proxy" && value.BenchmarkSymbol != "" && value.BenchmarkSymbol == map[string]string{"NASDAQ100_TR": "QQQ", "SP500_TR": "SPY"}[benchmark]
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
		if c.PurchaseStatus == "open" || c.PurchaseStatus == "limited" {
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
