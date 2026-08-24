package domain

import (
	"testing"
	"time"
)

func TestValidateMarketProfile(t *testing.T) {
	if err := ValidateMarketProfile(ExecutionDirectUS, "USD", "", true); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMarketProfile(ExecutionQDII, "CNY", "", false); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMarketProfile(ExecutionDirectUS, "CNY", "", true); err == nil {
		t.Fatal("expected currency mismatch")
	}
	if err := ValidateMarketProfile(ExecutionStockConnect, "CNY", "", true); err == nil {
		t.Fatal("stock connect must not use personal FX quota")
	}
}

func TestSystemPresetMarketProfiles(t *testing.T) {
	if got := len(SystemPresets()); got != 13 {
		t.Fatalf("system presets=%d want 13", got)
	}
	tests := []struct {
		name, mode, currency, benchmark string
		quota                           bool
	}{
		{"现金", ExecutionLocal, "CNY", "", false},
		{"纳指（QDII）", ExecutionQDII, "CNY", "NASDAQ100_TR", false},
		{"港股", ExecutionDirectHK, "HKD", "", true},
		{"美股", ExecutionDirectUS, "USD", "", true},
	}
	for _, tt := range tests {
		profile, ok := PresetMarketProfile(tt.name)
		if !ok || profile.ExecutionMode != tt.mode || profile.SettlementCurrency != tt.currency || profile.BenchmarkID != tt.benchmark || profile.FXQuotaRelevant != tt.quota {
			t.Fatalf("preset %q = %#v, %v", tt.name, profile, ok)
		}
	}
	if _, ok := PresetMarketProfile("苹果"); ok {
		t.Fatal("custom target must not receive a system profile")
	}
}

func TestValidateBenchmarkMapping(t *testing.T) {
	from := time.Now()
	if err := ValidateBenchmarkMapping("adjusted_nav", "total_return", from, nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBenchmarkMapping("unit_nav", "price", from, nil); err == nil {
		t.Fatal("expected incompatible basis")
	}
	to := from.Add(-time.Hour)
	if err := ValidateBenchmarkMapping("adjusted_nav", "total_return", from, &to); err == nil {
		t.Fatal("expected invalid period")
	}
}
