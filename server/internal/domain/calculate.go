package domain

import (
	"fmt"
	"regexp"
	"sort"
)

var monthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type allocationRemainder struct {
	index     int
	remainder int64
	sortOrder int
	id        string
}

func Calculate(plan Plan, input MonthlyInput, destinations []Destination) (Calculation, error) {
	if plan.Status != PlanActive {
		return Calculation{}, fmt.Errorf("%w: plan is not active", ErrInvalidPlan)
	}
	if err := ValidatePlan(plan); err != nil {
		return Calculation{}, err
	}
	if !monthPattern.MatchString(input.Month) {
		return Calculation{}, fmt.Errorf("%w: month must use YYYY-MM", ErrInvalidInput)
	}
	if !ValidateMoney(input.IncomeCents) || input.ContributionBPS < 0 || input.ContributionBPS > BasisPointTotal {
		return Calculation{}, fmt.Errorf("%w: invalid income or contribution rate", ErrInvalidInput)
	}
	expenseTotal := int64(0)
	for _, expense := range input.Expenses {
		if !ValidateMoney(expense.AmountCents) {
			return Calculation{}, fmt.Errorf("%w: invalid expense amount", ErrInvalidInput)
		}
		expenseTotal += expense.AmountCents
		if expenseTotal > MaxMoneyCents {
			return Calculation{}, fmt.Errorf("%w: expense total is too large", ErrInvalidInput)
		}
	}
	surplus := input.IncomeCents - expenseTotal
	investable := surplus - plan.ReserveCents
	if investable < 0 {
		investable = 0
	}
	rawRecommended := investable * int64(input.ContributionBPS) / BasisPointTotal
	recommended := rawRecommended / plan.RoundingUnitCents * plan.RoundingUnitCents

	active := make([]Destination, 0, len(destinations))
	for _, destination := range destinations {
		if destination.Active && !destination.Archived {
			active = append(active, destination)
		}
	}
	if len(active) == 0 {
		return Calculation{}, fmt.Errorf("%w: at least one active destination is required", ErrInvalidPlan)
	}
	units := recommended / plan.RoundingUnitCents
	allocations := make([]DestinationAllocation, len(active))
	remainders := make([]allocationRemainder, len(active))
	allocatedUnits := int64(0)
	for i, destination := range active {
		numerator := units * int64(destination.AllocationBPS)
		baseUnits := numerator / BasisPointTotal
		allocatedUnits += baseUnits
		allocations[i] = DestinationAllocation{
			DestinationID:    destination.ID,
			Name:             destination.Name,
			SortOrder:        destination.SortOrder,
			AllocationBPS:    destination.AllocationBPS,
			RecommendedCents: baseUnits * plan.RoundingUnitCents,
		}
		remainders[i] = allocationRemainder{
			index: i, remainder: numerator % BasisPointTotal, sortOrder: destination.SortOrder, id: destination.ID,
		}
	}
	sort.SliceStable(remainders, func(i, j int) bool {
		if remainders[i].remainder != remainders[j].remainder {
			return remainders[i].remainder > remainders[j].remainder
		}
		if remainders[i].sortOrder != remainders[j].sortOrder {
			return remainders[i].sortOrder < remainders[j].sortOrder
		}
		return remainders[i].id < remainders[j].id
	})
	for remaining := units - allocatedUnits; remaining > 0; remaining-- {
		target := remainders[int(units-allocatedUnits-remaining)%len(remainders)].index
		allocations[target].RecommendedCents += plan.RoundingUnitCents
	}
	return Calculation{
		Month:                 input.Month,
		IncomeCents:           input.IncomeCents,
		ExpenseTotalCents:     expenseTotal,
		ReserveCents:          plan.ReserveCents,
		SurplusCents:          surplus,
		InvestableBaseCents:   investable,
		ContributionBPS:       input.ContributionBPS,
		RecommendedTotalCents: recommended,
		RoundingUnitCents:     plan.RoundingUnitCents,
		Allocations:           allocations,
		Status:                DeriveExecutionStatus(recommended, 0),
	}, nil
}
