package service

import (
	"math"
	"testing"
	"time"

	"investplanner/server/internal/marketdata"
)

func TestTrackingErrorRequiresAdjustedCompatibleAndEnoughData(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	fund := make([]marketdata.SeriesPoint, 0, 150)
	benchmark := make([]marketdata.SeriesPoint, 0, 150)
	for i := 0; i < 150; i++ {
		date := now.AddDate(0, 0, -149+i)
		base := 100 * math.Pow(1.0005, float64(i))
		fund = append(fund, marketdata.SeriesPoint{Date: date, Value: base * (1 + 0.0001*math.Sin(float64(i)))})
		benchmark = append(benchmark, marketdata.SeriesPoint{Date: date, Value: base})
	}
	value, count, _, reason := TrackingError(fund, "adjusted_nav", benchmark, "total_return", now)
	if reason != "" || count < 120 || value <= 0 {
		t.Fatalf("value=%f count=%d reason=%s", value, count, reason)
	}
	if _, _, _, reason := TrackingError(fund, "unit_nav", benchmark, "price", now); reason != "收益口径不兼容" {
		t.Fatal(reason)
	}
	if value, count, _, reason := TrackingError(fund, "cumulative_nav_proxy", benchmark, "adjusted_close_cny_proxy", now); reason != "" || count < 120 || value <= 0 {
		t.Fatalf("proxy value=%f count=%d reason=%s", value, count, reason)
	}
	if _, _, _, reason := TrackingError(fund, "adjusted_close_cny", benchmark, "adjusted_close_cny_proxy", now); reason != "收益口径不兼容" {
		t.Fatalf("generic alignment must reject ETF basis, reason=%s", reason)
	}
	if _, _, _, reason := TrackingError(fund[:50], "adjusted_nav", benchmark[:50], "total_return", now); reason != "共同观测日不足 120 个" {
		t.Fatal(reason)
	}
}

func TestQDIIETFTrackingErrorUsesPreviousCompletedUSSession(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	benchmark := make([]marketdata.SeriesPoint, 0, 151)
	fund := make([]marketdata.SeriesPoint, 0, 150)
	for i := 0; i < 151; i++ {
		date := now.AddDate(0, 0, -151+i)
		base := 100 * math.Pow(1.0005, float64(i))
		benchmark = append(benchmark, marketdata.SeriesPoint{Date: date, Value: base})
		if i > 0 {
			fund = append(fund, marketdata.SeriesPoint{Date: date.AddDate(0, 0, 1), Value: base * (1 + 0.0001*math.Sin(float64(i)))})
		}
	}
	value, count, through, reason := marketdata.AnnualizedQDIIETFTrackingError(fund, "adjusted_close_cny", benchmark, "adjusted_close_cny_proxy", now)
	if reason != "" || count < 120 || value <= 0 || value > 0.01 || !through.Before(now) {
		t.Fatalf("value=%f count=%d through=%s reason=%s", value, count, through, reason)
	}
}

func TestCandidateOrderingKeepsUnknownDistinctFromZero(t *testing.T) {
	limit := int64(10000)
	known := QDIICandidate{FundCode: "A", PurchaseStatus: "open", DailyLimitCents: knownValue(limit, "10000", "p", "s", time.Now(), "fresh")}
	unknown := QDIICandidate{FundCode: "B", PurchaseStatus: "unknown", DailyLimitCents: unknownValue[int64]("unknown")}
	suspended := QDIICandidate{FundCode: "C", PurchaseStatus: "suspended", ExecutableCapacityCents: knownValue(int64(0), "0", "p", "s", time.Now(), "fresh")}
	if !betterOTC(known, unknown) || !betterOTC(unknown, suspended) {
		t.Fatal("unexpected ranking")
	}
	if suspended.ExecutableCapacityCents.Value == nil || *suspended.ExecutableCapacityCents.Value != 0 {
		t.Fatal("suspended capacity must be real zero")
	}
	if unknown.DailyLimitCents.Value != nil || unknown.DailyLimitCents.Display != "--" {
		t.Fatal("unknown must remain missing")
	}
}

func TestETFRequiresIOPVAndChoosesLowestAbsolutePremium(t *testing.T) {
	a, b := 0.03, -0.01
	first := QDIICandidate{PremiumRate: knownValue(a, "3%", "p", "s", time.Now(), "fresh")}
	second := QDIICandidate{PremiumRate: knownValue(b, "-1%", "p", "s", time.Now(), "fresh")}
	if !betterETF(second, first) {
		t.Fatal("expected lower absolute premium")
	}
	missing := QDIICandidate{PremiumRate: unknownValue[float64]("missing")}
	if betterETF(missing, first) {
		t.Fatal("missing IOPV/premium cannot win")
	}
}
