package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	ExecutionUnknown      = "unknown"
	ExecutionLocal        = "local"
	ExecutionQDII         = "qdii"
	ExecutionDirectUS     = "direct_us"
	ExecutionDirectHK     = "direct_hk"
	ExecutionStockConnect = "stock_connect"
)

var validExecutionModes = map[string]bool{
	ExecutionUnknown: true, ExecutionLocal: true, ExecutionQDII: true,
	ExecutionDirectUS: true, ExecutionDirectHK: true, ExecutionStockConnect: true,
}

type MarketProfile struct {
	ExecutionMode      string
	SettlementCurrency string
	BenchmarkID        string
	FXQuotaRelevant    bool
}

type SystemPreset struct {
	Name    string
	Aliases []string
	Profile MarketProfile
}

var systemPresets = []SystemPreset{
	{Name: "现金", Profile: MarketProfile{ExecutionMode: ExecutionLocal, SettlementCurrency: "CNY"}},
	{Name: "债券类基金", Profile: MarketProfile{ExecutionMode: ExecutionLocal, SettlementCurrency: "CNY"}},
	{Name: "纳斯达克100指数（QDII）", Aliases: []string{"纳指（QDII）"}, Profile: MarketProfile{ExecutionMode: ExecutionQDII, SettlementCurrency: "CNY", BenchmarkID: "NASDAQ100_TR"}},
	{Name: "标普500指数（QDII）", Aliases: []string{"标普（QDII）"}, Profile: MarketProfile{ExecutionMode: ExecutionQDII, SettlementCurrency: "CNY", BenchmarkID: "SP500_TR"}},
	{Name: "红利低波", Profile: MarketProfile{ExecutionMode: ExecutionLocal, SettlementCurrency: "CNY"}},
	{Name: "A股主动基金", Profile: MarketProfile{ExecutionMode: ExecutionLocal, SettlementCurrency: "CNY"}},
	{Name: "沪深300指数基金", Aliases: []string{"A股指数基金（如沪深300）"}, Profile: MarketProfile{ExecutionMode: ExecutionLocal, SettlementCurrency: "CNY"}},
	{Name: "中证500指数基金", Profile: MarketProfile{ExecutionMode: ExecutionLocal, SettlementCurrency: "CNY"}},
	{Name: "A股", Profile: MarketProfile{ExecutionMode: ExecutionLocal, SettlementCurrency: "CNY"}},
	{Name: "港股", Profile: MarketProfile{ExecutionMode: ExecutionDirectHK, SettlementCurrency: "HKD", FXQuotaRelevant: true}},
	{Name: "美股", Profile: MarketProfile{ExecutionMode: ExecutionDirectUS, SettlementCurrency: "USD", FXQuotaRelevant: true}},
	{Name: "纳斯达克100指数（美股）", Aliases: []string{"纳指（美股）"}, Profile: MarketProfile{ExecutionMode: ExecutionDirectUS, SettlementCurrency: "USD", FXQuotaRelevant: true}},
	{Name: "标普500指数（美股）", Aliases: []string{"标普（美股）"}, Profile: MarketProfile{ExecutionMode: ExecutionDirectUS, SettlementCurrency: "USD", FXQuotaRelevant: true}},
}

func SystemPresets() []SystemPreset {
	result := make([]SystemPreset, len(systemPresets))
	for i, preset := range systemPresets {
		result[i] = preset
		result[i].Aliases = append([]string(nil), preset.Aliases...)
	}
	return result
}

func PresetMarketProfile(name string) (MarketProfile, bool) {
	name = strings.TrimSpace(name)
	for _, preset := range systemPresets {
		if name == preset.Name {
			return preset.Profile, true
		}
		for _, alias := range preset.Aliases {
			if name == alias {
				return preset.Profile, true
			}
		}
	}
	return MarketProfile{}, false
}

func ValidateMarketProfile(mode, currency, benchmarkID string, quotaRelevant bool) error {
	if !validExecutionModes[mode] {
		return fmt.Errorf("%w: invalid execution mode", ErrInvalidPlan)
	}
	wantCurrency := map[string]string{
		ExecutionLocal: "CNY", ExecutionQDII: "CNY", ExecutionStockConnect: "CNY",
		ExecutionDirectUS: "USD", ExecutionDirectHK: "HKD", ExecutionUnknown: "unknown",
	}[mode]
	if currency != wantCurrency {
		return fmt.Errorf("%w: settlement currency is incompatible with execution mode", ErrInvalidPlan)
	}
	if quotaRelevant != (mode == ExecutionDirectUS || mode == ExecutionDirectHK) {
		return fmt.Errorf("%w: fx quota relevance is incompatible with execution mode", ErrInvalidPlan)
	}
	if mode == ExecutionQDII && strings.TrimSpace(benchmarkID) == "" {
		// A QDII destination may be saved before mapping, but the assistant will
		// remain unavailable. This is valid configuration, not guessed data.
		return nil
	}
	return nil
}

func ValidateBenchmarkMapping(fundBasis, benchmarkBasis string, from time.Time, to *time.Time) error {
	if from.IsZero() || (to != nil && !to.After(from)) {
		return fmt.Errorf("%w: invalid mapping effective period", ErrInvalidInput)
	}
	compatible := fundBasis == "adjusted_nav" && benchmarkBasis == "total_return"
	if !compatible {
		return fmt.Errorf("%w: incompatible return basis", ErrInvalidInput)
	}
	return nil
}
