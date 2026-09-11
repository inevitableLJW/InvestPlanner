package service

import (
	"testing"
	"time"

	"investplanner/server/internal/database"
	"investplanner/server/internal/domain"
)

func TestFXCurrencyClassification(t *testing.T) {
	cases := []struct {
		name        string
		destination database.PlanDestination
		want        string
	}{{"USD", database.PlanDestination{ExecutionMode: domain.ExecutionDirectUS, SettlementCurrency: "USD", FXQuotaRelevant: true}, "USD"}, {"HKD", database.PlanDestination{ExecutionMode: domain.ExecutionDirectHK, SettlementCurrency: "HKD", FXQuotaRelevant: true}, "HKD"}, {"QDII", database.PlanDestination{ExecutionMode: domain.ExecutionQDII, SettlementCurrency: "CNY"}, ""}, {"stock connect", database.PlanDestination{ExecutionMode: domain.ExecutionStockConnect, SettlementCurrency: "CNY"}, ""}, {"unknown", database.PlanDestination{ExecutionMode: domain.ExecutionUnknown, SettlementCurrency: "unknown"}, ""}, {"wrong currency", database.PlanDestination{ExecutionMode: domain.ExecutionDirectHK, SettlementCurrency: "USD", FXQuotaRelevant: true}, ""}}
	for _, tt := range cases {
		if got := fxCurrencyFor(tt.destination); got != tt.want {
			t.Errorf("%s=%q want %q", tt.name, got, tt.want)
		}
	}
}
func TestForecastMonths(t *testing.T) {
	now := time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)
	rolling := forecastMonths(now, "rolling_12")
	if rolling[0] != "2027-01" || rolling[11] != "2027-12" {
		t.Fatal(rolling)
	}
	next := forecastMonths(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), "next_calendar_year")
	if next[0] != "2027-01" || next[11] != "2027-12" {
		t.Fatal(next)
	}
}

func TestFXUnknownSummaryIsNotZero(t *testing.T) {
	rate := FXRateInfo{CNYPerUnit: 7}
	unknown := fxSummary("USD", 0, false, rate)
	if unknown.CNYCents != nil || unknown.ForeignAmount != nil {
		t.Fatal(unknown)
	}
	zero := fxSummary("USD", 0, true, rate)
	if zero.CNYCents == nil || zero.ForeignAmount == nil || *zero.ForeignAmount != 0 {
		t.Fatal(zero)
	}
	missingRate := fxSummary("USD", 70000, true, FXRateInfo{})
	if missingRate.CNYCents == nil || missingRate.ForeignAmount != nil {
		t.Fatal(missingRate)
	}
}

func TestFXFreshFetchDoesNotHideOldOrFutureObservation(t *testing.T) {
	now := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		observed time.Time
		want     string
	}{{now.AddDate(0, 0, -1), "fresh"}, {now.AddDate(0, 0, -30), "stale"}, {now.AddDate(0, 0, 1), "unavailable"}, {time.Time{}, "unavailable"}} {
		if got := fxRateFreshness(database.MarketDataSnapshot{ObservedAt: tc.observed, FetchedAt: now}, now); got != tc.want {
			t.Fatalf("%v: %s", tc, got)
		}
	}
}
