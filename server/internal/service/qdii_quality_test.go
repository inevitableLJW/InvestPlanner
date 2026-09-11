package service

import (
	"investplanner/server/internal/database"
	"investplanner/server/internal/marketdata"
	"testing"
	"time"
)

func TestPurchaseSnapshotsAreValidatedOnRead(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	zero, limit, minimum := int64(0), int64(1000), int64(100)
	snap := database.MarketDataSnapshot{Provider: "aktools", ObservedAt: now, FetchedAt: now}
	for _, tc := range []struct {
		name, status, raw, fresh string
		limit                    *int64
		wantStatus               string
		wantCapacity             bool
	}{
		{"upstream zero", "limited", "限大额", "fresh", &zero, "limited", false},
		{"legacy zero", "open", "", "fresh", &zero, "unknown", false},
		{"legacy lossy status", "suspended", "", "fresh", &limit, "unknown", false},
		{"temporary suspension", "suspended", "暂停申购", "fresh", &limit, "suspended", false},
		{"stale open", "open", "开放申购", "stale", &limit, "unknown", false},
		{"limited positive", "limited", "限大额", "fresh", &limit, "limited", true},
		{"not sold", "not_sold", "该渠道未开通代销", "fresh", &limit, "not_sold", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := QDIICandidate{RemainingNeedCents: 200000}
			applyPurchaseSnapshot(&c, purchaseSnapshot{Status: tc.status, RawStatus: tc.raw, DailyLimitCents: tc.limit, MinimumCents: &minimum}, snap, now, tc.fresh)
			if c.PurchaseStatus != tc.wantStatus || (c.ExecutableCapacityCents.Value != nil) != tc.wantCapacity {
				t.Fatalf("%+v", c)
			}
			if tc.limit == &zero && c.DailyLimitCents.Value != nil {
				t.Fatal("zero must not become allowance")
			}
			if tc.wantCapacity && (*c.ExecutableCapacityCents.Value != 18000 || c.RemainingPurchaseDays.Reason == "") {
				t.Fatalf("bad estimate: %+v", c)
			}
		})
	}
}

func TestCachedTrackingRejectsOldAlgorithmAndInvalidMetadata(t *testing.T) {
	now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	good := marketdata.TrackingErrorResult{MethodVersion: marketdata.TrackingMethodVersion, Value: .02, Count: 234, Through: now.AddDate(0, 0, -3), Basis: "fund_nav_proxy", BenchmarkSymbol: "QQQ"}
	if !validCachedTracking(good, "NASDAQ100_TR", now) {
		t.Fatal("valid cache rejected")
	}
	for _, alter := range []func(*marketdata.TrackingErrorResult){func(v *marketdata.TrackingErrorResult) { v.MethodVersion = 0 }, func(v *marketdata.TrackingErrorResult) { v.Count = 119 }, func(v *marketdata.TrackingErrorResult) { v.Through = now.AddDate(0, 0, 1) }, func(v *marketdata.TrackingErrorResult) { v.Through = now.AddDate(0, 0, -15) }, func(v *marketdata.TrackingErrorResult) { v.BenchmarkSymbol = "SPY" }} {
		bad := good
		alter(&bad)
		if validCachedTracking(bad, "NASDAQ100_TR", now) {
			t.Fatal("invalid cache accepted")
		}
	}
}
