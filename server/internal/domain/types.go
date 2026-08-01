package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	BasisPointTotal = 10_000
	MaxMoneyCents   = int64(9_000_000_000_000)
)

type PlanStatus string

const (
	PlanDraft    PlanStatus = "draft"
	PlanActive   PlanStatus = "active"
	PlanArchived PlanStatus = "archived"
)

type Destination struct {
	ID            string
	Name          string
	Active        bool
	Archived      bool
	SortOrder     int
	AllocationBPS int
}

type Plan struct {
	ID                     string
	UserID                 string
	Name                   string
	Status                 PlanStatus
	DefaultContributionBPS int
	ReserveCents           int64
	RoundingUnitCents      int64
	Version                int
	Destinations           []Destination
}

type ExpenseInput struct {
	SourceID    string
	SourceName  string
	AmountCents int64
}

type MonthlyInput struct {
	Month           string
	IncomeCents     int64
	ContributionBPS int
	Expenses        []ExpenseInput
	Note            string
}

type DestinationAllocation struct {
	DestinationID    string
	Name             string
	SortOrder        int
	AllocationBPS    int
	RecommendedCents int64
	ActualCents      int64
}

type Calculation struct {
	Month                 string
	IncomeCents           int64
	ExpenseTotalCents     int64
	ReserveCents          int64
	SurplusCents          int64
	InvestableBaseCents   int64
	ContributionBPS       int
	RecommendedTotalCents int64
	RoundingUnitCents     int64
	Allocations           []DestinationAllocation
	Status                ExecutionStatus
}

type ExecutionStatus string

const (
	StatusNoContribution ExecutionStatus = "not_required"
	StatusNotStarted     ExecutionStatus = "not_started"
	StatusPartial        ExecutionStatus = "partial"
	StatusComplete       ExecutionStatus = "complete"
)

type MonthlySummary struct {
	Month            string
	RecommendedCents int64
	ActualCents      int64
}

type DestinationSummary struct {
	Name        string
	ActualCents int64
}

type PlanStats struct {
	Monthly               []MonthlySummary
	RecommendedTotalCents int64
	ActualTotalCents      int64
	CompletionRate        *float64
	Destinations          []DestinationSummary
}

var (
	ErrInvalidPlan  = errors.New("invalid plan")
	ErrInvalidInput = errors.New("invalid monthly input")
)

func ValidateMoney(value int64) bool {
	return value >= 0 && value <= MaxMoneyCents
}

func ValidatePlan(plan Plan) error {
	if strings.TrimSpace(plan.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidPlan)
	}
	if plan.DefaultContributionBPS < 0 || plan.DefaultContributionBPS > BasisPointTotal {
		return fmt.Errorf("%w: contribution rate must be between 0 and 10000", ErrInvalidPlan)
	}
	if !ValidateMoney(plan.ReserveCents) || plan.RoundingUnitCents <= 0 || plan.RoundingUnitCents > MaxMoneyCents {
		return fmt.Errorf("%w: invalid reserve or rounding unit", ErrInvalidPlan)
	}
	active := make([]Destination, 0, len(plan.Destinations))
	names := map[string]struct{}{}
	total := 0
	for _, destination := range plan.Destinations {
		if !destination.Active || destination.Archived {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(destination.Name))
		if name == "" {
			return fmt.Errorf("%w: destination name is required", ErrInvalidPlan)
		}
		if _, exists := names[name]; exists {
			return fmt.Errorf("%w: duplicate active destination name", ErrInvalidPlan)
		}
		names[name] = struct{}{}
		if destination.AllocationBPS < 0 || destination.AllocationBPS > BasisPointTotal {
			return fmt.Errorf("%w: invalid destination allocation", ErrInvalidPlan)
		}
		total += destination.AllocationBPS
		active = append(active, destination)
	}
	if len(active) == 0 {
		return fmt.Errorf("%w: at least one destination is required", ErrInvalidPlan)
	}
	if total != BasisPointTotal {
		return fmt.Errorf("%w: active destination allocations must total 10000", ErrInvalidPlan)
	}
	return nil
}

func DeriveExecutionStatus(recommended, actual int64) ExecutionStatus {
	switch {
	case recommended == 0 && actual == 0:
		return StatusNoContribution
	case recommended > 0 && actual == 0:
		return StatusNotStarted
	case actual < recommended:
		return StatusPartial
	default:
		return StatusComplete
	}
}

func BuildStats(months []Calculation) PlanStats {
	stats := PlanStats{Monthly: make([]MonthlySummary, 0, len(months))}
	destinations := map[string]int64{}
	for _, month := range months {
		actual := int64(0)
		for _, allocation := range month.Allocations {
			actual += allocation.ActualCents
			destinations[allocation.Name] += allocation.ActualCents
		}
		stats.Monthly = append(stats.Monthly, MonthlySummary{
			Month: month.Month, RecommendedCents: month.RecommendedTotalCents, ActualCents: actual,
		})
		stats.RecommendedTotalCents += month.RecommendedTotalCents
		stats.ActualTotalCents += actual
	}
	sort.Slice(stats.Monthly, func(i, j int) bool { return stats.Monthly[i].Month < stats.Monthly[j].Month })
	for name, amount := range destinations {
		stats.Destinations = append(stats.Destinations, DestinationSummary{Name: name, ActualCents: amount})
	}
	sort.Slice(stats.Destinations, func(i, j int) bool {
		if stats.Destinations[i].ActualCents == stats.Destinations[j].ActualCents {
			return stats.Destinations[i].Name < stats.Destinations[j].Name
		}
		return stats.Destinations[i].ActualCents > stats.Destinations[j].ActualCents
	})
	if stats.RecommendedTotalCents > 0 {
		rate := float64(stats.ActualTotalCents) / float64(stats.RecommendedTotalCents)
		stats.CompletionRate = &rate
	}
	return stats
}
