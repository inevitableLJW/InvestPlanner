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
func TestComplianceNoticeDoesNotClaimInvestmentPermission(t *testing.T) {
	if FXComplianceNotice == "" {
		t.Fatal("missing notice")
	}
}
