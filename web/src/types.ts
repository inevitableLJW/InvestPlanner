export type PlanStatus = 'draft' | 'active' | 'archived'

export interface User { id: string; username: string }
export interface Destination { id?: string; name: string; active: boolean; archived: boolean; sortOrder: number; allocationBps: number; version?: number }
export interface Plan {
  id: string; name: string; status: PlanStatus; defaultContributionBps: number
  reserveCents: number; roundingUnitCents: number; version: number
  destinations: Destination[]; summary?: PlanStats
}
export interface ExpenseSource { id: string; name: string; sortOrder: number; active: boolean }
export interface Expense { id?: string; sourceId: string; sourceName: string; amountCents: number; sortOrder: number }
export interface Allocation {
  id: string; destinationId: string; name: string; sortOrder: number; allocationBps: number
  recommendedCents: number; actualCents: number; differenceCents: number
}
export interface MonthRecord {
  id: string; planId: string; month: string; incomeCents: number; expenseTotalCents: number
  reserveCents: number; surplusCents: number; investableBaseCents: number; contributionBps: number
  recommendedTotalCents: number; roundingUnitCents: number; actualTotalCents: number
  status: 'not_required' | 'not_started' | 'partial' | 'complete'; note: string; version: number
  expenses: Expense[]; allocations: Allocation[]
}
export interface PlanStats {
  Monthly?: Array<{ Month: string; RecommendedCents: number; ActualCents: number }>
  RecommendedTotalCents: number; ActualTotalCents: number; CompletionRate: number | null
  Destinations?: Array<{ Name: string; ActualCents: number }>
}
