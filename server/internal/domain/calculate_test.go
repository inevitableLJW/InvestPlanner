package domain

import (
	"testing"
)

func activePlan(destinations ...Destination) Plan {
	return Plan{
		ID: "plan-1", Name: "长期计划", Status: PlanActive,
		DefaultContributionBPS: 8000, ReserveCents: 200_000,
		RoundingUnitCents: 10_000, Destinations: destinations,
	}
}

func TestValidatePlanDestinations(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		ok   bool
	}{
		{"zero destinations", activePlan(), false},
		{"one destination", activePlan(Destination{ID: "cash", Name: "现金", Active: true, AllocationBPS: 10000}), true},
		{"subset defaults", activePlan(
			Destination{ID: "fund", Name: "支付宝基金", Active: true, AllocationBPS: 6000},
			Destination{ID: "cash", Name: "现金", Active: true, AllocationBPS: 4000},
			Destination{ID: "a", Name: "A股", Active: false, AllocationBPS: 0},
		), true},
		{"wrong total", activePlan(Destination{ID: "cash", Name: "现金", Active: true, AllocationBPS: 9000}), false},
		{"duplicate active names", activePlan(
			Destination{ID: "one", Name: "现金", Active: true, AllocationBPS: 5000},
			Destination{ID: "two", Name: " 现金 ", Active: true, AllocationBPS: 5000},
		), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidatePlan(test.plan)
			if test.ok && err != nil {
				t.Fatalf("expected valid plan: %v", err)
			}
			if !test.ok && err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestCalculateOneDestination(t *testing.T) {
	plan := activePlan(Destination{ID: "cash", Name: "现金", Active: true, AllocationBPS: 10000})
	result, err := Calculate(plan, MonthlyInput{
		Month: "2026-07", IncomeCents: 2_000_000, ContributionBPS: 8000,
		Expenses: []ExpenseInput{{SourceName: "支付宝", AmountCents: 1_000_000}},
	}, plan.Destinations)
	if err != nil {
		t.Fatal(err)
	}
	if result.InvestableBaseCents != 800_000 {
		t.Fatalf("investable base = %d", result.InvestableBaseCents)
	}
	if result.RecommendedTotalCents != 640_000 {
		t.Fatalf("recommended = %d", result.RecommendedTotalCents)
	}
	if len(result.Allocations) != 1 || result.Allocations[0].RecommendedCents != 640_000 {
		t.Fatalf("unexpected allocations: %#v", result.Allocations)
	}
}

func TestCalculateLargestRemainderUsesStableOrder(t *testing.T) {
	plan := activePlan(
		Destination{ID: "b", Name: "港美股", Active: true, AllocationBPS: 3333, SortOrder: 2},
		Destination{ID: "a", Name: "A股", Active: true, AllocationBPS: 3333, SortOrder: 1},
		Destination{ID: "c", Name: "现金", Active: true, AllocationBPS: 3334, SortOrder: 3},
	)
	plan.ReserveCents = 0
	plan.RoundingUnitCents = 100
	result, err := Calculate(plan, MonthlyInput{Month: "2026-07", IncomeCents: 1000, ContributionBPS: 10000}, plan.Destinations)
	if err != nil {
		t.Fatal(err)
	}
	total := int64(0)
	byName := map[string]int64{}
	for _, allocation := range result.Allocations {
		total += allocation.RecommendedCents
		byName[allocation.Name] = allocation.RecommendedCents
	}
	if total != 1000 {
		t.Fatalf("allocation total = %d", total)
	}
	if byName["现金"] != 400 || byName["A股"] != 300 || byName["港美股"] != 300 {
		t.Fatalf("unexpected largest remainder allocation: %#v", byName)
	}
}

func TestCalculateInsufficientSurplus(t *testing.T) {
	plan := activePlan(Destination{ID: "cash", Name: "现金", Active: true, AllocationBPS: 10000})
	result, err := Calculate(plan, MonthlyInput{
		Month: "2026-07", IncomeCents: 100_000, ContributionBPS: 10000,
		Expenses: []ExpenseInput{{SourceName: "微信", AmountCents: 150_000}},
	}, plan.Destinations)
	if err != nil {
		t.Fatal(err)
	}
	if result.InvestableBaseCents != 0 || result.RecommendedTotalCents != 0 || result.Status != StatusNoContribution {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestBuildStatsIncludesCashAndOverContribution(t *testing.T) {
	stats := BuildStats([]Calculation{
		{Month: "2026-07", RecommendedTotalCents: 1000, Allocations: []DestinationAllocation{{Name: "现金", ActualCents: 1200}}},
		{Month: "2026-06", RecommendedTotalCents: 1000, Allocations: []DestinationAllocation{{Name: "自定义", ActualCents: 1000}}},
	})
	if stats.ActualTotalCents != 2200 || stats.RecommendedTotalCents != 2000 {
		t.Fatalf("unexpected totals: %#v", stats)
	}
	if stats.CompletionRate == nil || *stats.CompletionRate != 1.1 {
		t.Fatalf("unexpected completion rate: %#v", stats.CompletionRate)
	}
	if stats.Monthly[0].Month != "2026-06" || len(stats.Destinations) != 2 {
		t.Fatalf("unexpected sorting: %#v", stats)
	}
}

func TestBuildStatsNoRecommendation(t *testing.T) {
	stats := BuildStats([]Calculation{{Month: "2026-07"}})
	if stats.CompletionRate != nil {
		t.Fatal("completion rate must be nil when recommendation total is zero")
	}
}
