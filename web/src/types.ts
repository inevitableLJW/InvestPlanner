export type PlanStatus = 'draft' | 'active' | 'archived'

export interface User { id: string; username: string }
export type ExecutionMode = 'unknown' | 'local' | 'qdii' | 'direct_us' | 'direct_hk' | 'stock_connect'
export interface Destination { id?: string; name: string; active: boolean; archived: boolean; sortOrder: number; allocationBps: number; version?: number; executionMode?: ExecutionMode; settlementCurrency?: 'unknown' | 'CNY' | 'USD' | 'HKD'; benchmarkId?: string; fxQuotaRelevant?: boolean }
export interface Plan {
  id: string; name: string; status: PlanStatus; defaultContributionBps: number
  reserveCents: number; roundingUnitCents: number; version: number; deletable: boolean
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
export interface AssistantValue<T> { value: T | null; display: string; reason?: string; provider?: string; sourceUrl?: string; observedAt?: string; freshness: 'fresh' | 'stale' | 'unavailable' }
export interface QDIICandidate { fundCode: string; name: string; shareClass: string; venue: string; channel?: string; purchaseStatus?: string; purchaseStatusInfo?: AssistantValue<string>; nextOpenDate: AssistantValue<string>; dailyLimitCents: AssistantValue<number>; minimumCents: AssistantValue<number>; remainingNeedCents: number; remainingPurchaseDays: AssistantValue<number>; executableCapacityCents: AssistantValue<number>; trackingError: AssistantValue<number>; price: AssistantValue<number>; iopv: AssistantValue<number>; premiumRate: AssistantValue<number> }
export interface QDIIConfirmedOTCPlan { fundCode: string; dailyAmountCents: number | null; totalAmountCents: number | null }
export interface QDIIConfirmation { selectedOtcFundCodes: string[]; selectedOtcPlans: QDIIConfirmedOTCPlan[]; selectedEtfFundCode: string; selectedEtfAmountCents: number | null; confirmedAt: string }
export interface QDIIConfirmedFund { fundCode: string; name: string; shareClass?: string; dailyAmountCents: number | null; totalAmountCents: number | null }
export interface QDIIConfirmationSummary { destinationId: string; selectedOtcFunds: QDIIConfirmedFund[]; selectedEtfFund?: QDIIConfirmedFund; selectedEtfAmountCents: number | null; confirmedAt: string }
export interface QDIIResult { destinationId: string; benchmarkId: string; status: 'fresh' | 'stale' | 'unavailable'; reason?: string; otc: QDIICandidate[]; etfs: QDIICandidate[]; confirmation?: QDIIConfirmation }
export interface FXMonth { month: string; currency: 'USD' | 'HKD'; cnyCents: number | null; foreignAmount: number | null; usdEquivalent: number | null; cumulativeUsdEquivalent: number | null; details: Array<{ destinationId: string; name: string; cnyCents: number }>; reason?: string }
export interface FXForecast { window: 'rolling_12' | 'next_calendar_year'; status?: 'fresh' | 'stale' | 'unavailable'; reason?: string; currencies: Array<'USD' | 'HKD'>; months: FXMonth[]; summaries: Array<{ currency: 'USD' | 'HKD'; cnyCents: number | null; foreignAmount: number | null }>; rates: Array<{ currency: string; cnyPerUnit: number; provider: string; sourceUrl: string; observedAt: string; assumption: string; freshness: string }>; unclassifiedDestinationIds?: string[] }
