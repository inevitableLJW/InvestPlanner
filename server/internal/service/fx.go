package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"investplanner/server/internal/database"
	"investplanner/server/internal/domain"
	"investplanner/server/internal/repository"
)

const FXComplianceNotice = "年度便利化额度仅是资金规划参考，不是绝对购汇上限，也不代表或构成境外证券投资许可。"

type FXService struct {
	Store              *repository.Store
	Now                func() time.Time
	Enabled            bool
	QuotaUSD           float64
	QuotaSource        string
	QuotaEffectiveDate string
}
type FXDetail struct {
	DestinationID string `json:"destinationId"`
	Name          string `json:"name"`
	CNYCents      int64  `json:"cnyCents"`
}
type FXMonth struct {
	Month                   string     `json:"month"`
	Currency                string     `json:"currency"`
	CNYCents                *int64     `json:"cnyCents"`
	ForeignAmount           *float64   `json:"foreignAmount"`
	USDEquivalent           *float64   `json:"usdEquivalent"`
	CumulativeUSDEquivalent *float64   `json:"cumulativeUsdEquivalent"`
	Details                 []FXDetail `json:"details"`
	Reason                  string     `json:"reason,omitempty"`
}
type FXCurrencySummary struct {
	Currency      string   `json:"currency"`
	CNYCents      *int64   `json:"cnyCents"`
	ForeignAmount *float64 `json:"foreignAmount"`
}
type FXRateInfo struct {
	Currency   string    `json:"currency"`
	CNYPerUnit float64   `json:"cnyPerUnit"`
	Provider   string    `json:"provider"`
	SourceURL  string    `json:"sourceUrl"`
	ObservedAt time.Time `json:"observedAt"`
	Assumption string    `json:"assumption"`
	Freshness  string    `json:"freshness"`
}
type FXForecast struct {
	Window                     string              `json:"window"`
	Status                     string              `json:"status"`
	Reason                     string              `json:"reason,omitempty"`
	Currencies                 []string            `json:"currencies"`
	Months                     []FXMonth           `json:"months"`
	Summaries                  []FXCurrencySummary `json:"summaries"`
	Rates                      []FXRateInfo        `json:"rates"`
	QuotaUSD                   float64             `json:"quotaUsd"`
	QuotaSource                string              `json:"quotaSource"`
	QuotaEffectiveDate         string              `json:"quotaEffectiveDate"`
	ComplianceNotice           string              `json:"complianceNotice"`
	UnclassifiedDestinationIDs []string            `json:"unclassifiedDestinationIds,omitempty"`
}
type fxSnapshot struct {
	CNYPerUnit float64 `json:"cnyPerUnit"`
	Unit       string  `json:"unit"`
}

func (s *FXService) Forecast(userID, planID, window string, scenarios map[string]float64) (FXForecast, error) {
	if window != "next_calendar_year" && window != "rolling_12" {
		return FXForecast{}, fmt.Errorf("%w: 预测窗口无效", ErrValidation)
	}
	planModel, err := s.Store.GetPlan(userID, planID)
	if err != nil {
		return FXForecast{}, err
	}
	result := FXForecast{Window: window, Status: "fresh", Currencies: []string{}, Months: []FXMonth{}, Summaries: []FXCurrencySummary{}, Rates: []FXRateInfo{}, QuotaUSD: s.QuotaUSD, QuotaSource: s.QuotaSource, QuotaEffectiveDate: s.QuotaEffectiveDate, ComplianceNotice: FXComplianceNotice}
	if result.QuotaUSD == 0 {
		result.QuotaUSD = 50000
	}
	profiles := map[string]database.PlanDestination{}
	currencySet := map[string]bool{}
	for _, d := range planModel.Destinations {
		profiles[d.ID] = d
		if !d.Active || d.ArchivedAt != nil {
			continue
		}
		if currency := fxCurrencyFor(d); currency != "" {
			currencySet[currency] = true
		} else if d.ExecutionMode == domain.ExecutionUnknown {
			result.UnclassifiedDestinationIDs = append(result.UnclassifiedDestinationIDs, d.ID)
		}
	}
	for _, currency := range []string{"USD", "HKD"} {
		if currencySet[currency] {
			result.Currencies = append(result.Currencies, currency)
		}
	}
	if len(result.Currencies) == 0 {
		result.Status = "unavailable"
		result.Reason = "计划没有需个人换汇的海外直投标的"
		return result, nil
	}
	if !s.Enabled {
		result.Status = "unavailable"
		result.Reason = "年度换汇预测暂不可用"
		return result, nil
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	rates := map[string]FXRateInfo{}
	for _, currency := range []string{"USD", "HKD"} {
		if value, ok := scenarios[currency]; ok {
			if value <= 0 || value > 1000 {
				return FXForecast{}, fmt.Errorf("%w: 情景汇率必须大于 0 且不超过 1000", ErrValidation)
			}
			rates[currency] = FXRateInfo{Currency: currency, CNYPerUnit: value, Provider: "user_scenario", Assumption: "scenario", ObservedAt: now, Freshness: "fresh"}
			continue
		}
		var data fxSnapshot
		snap, err := s.Store.GetSnapshot("fx_rate", currency)
		if err != nil {
			snap, err = s.Store.GetSnapshot("fx_rate_fallback", currency)
		}
		if err == nil && json.Unmarshal(snap.NormalizedJSON, &data) == nil && data.CNYPerUnit > 0 {
			freshness := "fresh"
			if now.Sub(snap.FetchedAt) > 36*time.Hour {
				freshness = "stale"
			}
			rates[currency] = FXRateInfo{Currency: currency, CNYPerUnit: data.CNYPerUnit, Provider: snap.Provider, SourceURL: snap.SourceURL, ObservedAt: snap.ObservedAt, Assumption: "latest_midpoint_static", Freshness: freshness}
		}
	}
	for _, currency := range []string{"USD", "HKD"} {
		if rate, ok := rates[currency]; ok && (currencySet[currency] || currency == "USD") {
			result.Rates = append(result.Rates, rate)
		}
	}
	months := forecastMonths(now, window)
	records, err := s.Store.ListMonths(userID, planID)
	if err != nil {
		return FXForecast{}, err
	}
	var baseline *database.MonthlyRecord
	if len(records) > 0 {
		sort.Slice(records, func(i, j int) bool { return records[i].Month > records[j].Month })
		loaded, loadErr := s.Store.GetMonth(userID, planID, records[0].Month)
		if loadErr != nil {
			return FXForecast{}, loadErr
		}
		baseline = &loaded
	}
	totals := map[string]int64{}
	cumulative := 0.0
	cumulativeKnown := true
	for _, month := range months {
		byCurrency := map[string][]FXDetail{}
		known := baseline != nil
		if baseline != nil {
			expenses := []domain.ExpenseInput{}
			for _, e := range baseline.Expenses {
				expenses = append(expenses, domain.ExpenseInput{SourceName: e.SourceName, AmountCents: e.AmountCents})
			}
			calculation, calcErr := domain.Calculate(planToDomain(planModel), domain.MonthlyInput{Month: month, IncomeCents: baseline.IncomeCents, ContributionBPS: baseline.ContributionBPS, Expenses: expenses}, planToDomain(planModel).Destinations)
			if calcErr != nil {
				known = false
			} else {
				for _, a := range calculation.Allocations {
					profile, ok := profiles[a.DestinationID]
					if !ok || !profile.FXQuotaRelevant {
						continue
					}
					currency := fxCurrencyFor(profile)
					if currency != "" {
						byCurrency[currency] = append(byCurrency[currency], FXDetail{DestinationID: a.DestinationID, Name: a.Name, CNYCents: a.RecommendedCents})
					}
				}
			}
		}
		for _, currency := range result.Currencies {
			entry := FXMonth{Month: month, Currency: currency, Details: byCurrency[currency]}
			if !known {
				entry.Reason = "缺少可复用的权威月度建议基线"
				cumulativeKnown = false
				result.Months = append(result.Months, entry)
				continue
			}
			cny := int64(0)
			for _, d := range entry.Details {
				cny += d.CNYCents
			}
			entry.CNYCents = &cny
			totals[currency] += cny
			rate, hasRate := rates[currency]
			usdRate, hasUSD := rates["USD"]
			if hasRate {
				foreign := float64(cny) / 100 / rate.CNYPerUnit
				entry.ForeignAmount = &foreign
			}
			if hasUSD {
				usd := float64(cny) / 100 / usdRate.CNYPerUnit
				entry.USDEquivalent = &usd
				if cumulativeKnown {
					cumulative += usd
					copy := cumulative
					entry.CumulativeUSDEquivalent = &copy
				}
			} else {
				entry.Reason = "必要的 USD 参考汇率不可用"
				cumulativeKnown = false
			}
			result.Months = append(result.Months, entry)
		}
	}
	for _, currency := range result.Currencies {
		total := totals[currency]
		summary := FXCurrencySummary{Currency: currency, CNYCents: &total}
		if rate, ok := rates[currency]; ok {
			foreign := float64(total) / 100 / rate.CNYPerUnit
			summary.ForeignAmount = &foreign
		}
		result.Summaries = append(result.Summaries, summary)
	}
	return result, nil
}

func fxCurrencyFor(destination database.PlanDestination) string {
	if !destination.FXQuotaRelevant {
		return ""
	}
	if destination.ExecutionMode == domain.ExecutionDirectUS && destination.SettlementCurrency == "USD" {
		return "USD"
	}
	if destination.ExecutionMode == domain.ExecutionDirectHK && destination.SettlementCurrency == "HKD" {
		return "HKD"
	}
	return ""
}
func forecastMonths(now time.Time, window string) []string {
	year, month := now.Year(), now.Month()
	if window == "next_calendar_year" {
		year++
		month = time.January
	} else {
		month++
		if month > time.December {
			year++
			month = time.January
		}
	}
	out := make([]string, 12)
	for i := range out {
		date := time.Date(year, time.Month(int(month)+i), 1, 0, 0, 0, 0, now.Location())
		out[i] = date.Format("2006-01")
	}
	return out
}
