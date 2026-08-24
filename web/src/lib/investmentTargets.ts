import type { Destination, ExecutionMode } from '../types'

export const COMMON_INVESTMENT_TARGETS = [
  '现金',
  '债券类基金',
  '纳斯达克100指数（QDII）',
  '标普500指数（QDII）',
  '红利低波',
  'A股主动基金',
  '沪深300指数基金',
  '中证500指数基金',
  'A股',
  '港股',
  '美股',
  '纳斯达克100指数（美股）',
  '标普500指数（美股）',
] as const

export const CUSTOM_TARGET_VALUE = '__custom__'
export const ALLOCATION_PERCENT_OPTIONS = [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 60, 70, 75, 80, 90, 100]

export interface InvestmentTargetMarketProfile {
  executionMode: ExecutionMode
  settlementCurrency: 'unknown' | 'CNY' | 'USD' | 'HKD'
  benchmarkId: string
  fxQuotaRelevant: boolean
}

const localProfile: InvestmentTargetMarketProfile = { executionMode: 'local', settlementCurrency: 'CNY', benchmarkId: '', fxQuotaRelevant: false }
const PRESET_MARKET_PROFILES: Record<(typeof COMMON_INVESTMENT_TARGETS)[number], InvestmentTargetMarketProfile> = {
  '现金': localProfile,
  '债券类基金': localProfile,
  '纳斯达克100指数（QDII）': { executionMode: 'qdii', settlementCurrency: 'CNY', benchmarkId: 'NASDAQ100_TR', fxQuotaRelevant: false },
  '标普500指数（QDII）': { executionMode: 'qdii', settlementCurrency: 'CNY', benchmarkId: 'SP500_TR', fxQuotaRelevant: false },
  '红利低波': localProfile,
  'A股主动基金': localProfile,
  '沪深300指数基金': localProfile,
  '中证500指数基金': localProfile,
  'A股': localProfile,
  '港股': { executionMode: 'direct_hk', settlementCurrency: 'HKD', benchmarkId: '', fxQuotaRelevant: true },
  '美股': { executionMode: 'direct_us', settlementCurrency: 'USD', benchmarkId: '', fxQuotaRelevant: true },
  '纳斯达克100指数（美股）': { executionMode: 'direct_us', settlementCurrency: 'USD', benchmarkId: '', fxQuotaRelevant: true },
  '标普500指数（美股）': { executionMode: 'direct_us', settlementCurrency: 'USD', benchmarkId: '', fxQuotaRelevant: true },
}

const LEGACY_TARGET_NAMES: Record<string, string> = {
  '纳指（QDII）': '纳斯达克100指数（QDII）',
  '标普（QDII）': '标普500指数（QDII）',
  '纳指（美股）': '纳斯达克100指数（美股）',
  '标普（美股）': '标普500指数（美股）',
  'A股指数基金（如沪深300）': '沪深300指数基金',
}

export function normalizeInvestmentTargetName(name: string) {
  const trimmed = name.trim()
  return LEGACY_TARGET_NAMES[trimmed] ?? trimmed
}

export function isCommonInvestmentTarget(name: string) {
  return COMMON_INVESTMENT_TARGETS.includes(normalizeInvestmentTargetName(name) as (typeof COMMON_INVESTMENT_TARGETS)[number])
}

export function investmentTargetSelectValue(name: string) {
  const normalized = normalizeInvestmentTargetName(name)
  return isCommonInvestmentTarget(normalized) ? normalized : CUSTOM_TARGET_VALUE
}

export function presetMarketProfile(name: string): InvestmentTargetMarketProfile | undefined {
  const normalized = normalizeInvestmentTargetName(name)
  return PRESET_MARKET_PROFILES[normalized as (typeof COMMON_INVESTMENT_TARGETS)[number]]
}

export function withPresetMarketProfile(destination: Destination): Destination {
  const profile = presetMarketProfile(destination.name)
  return profile ? { ...destination, ...profile } : destination
}

export function allocationOptions(currentBps: number) {
  const currentPercent = currentBps / 100
  return [...new Set([...ALLOCATION_PERCENT_OPTIONS, currentPercent])].sort((a, b) => a - b)
}

export function activeAllocationTotal(destinations: Destination[]) {
  return destinations.filter(item => item.active && !item.archived).reduce((total, item) => total + item.allocationBps, 0)
}

export function fillRemainingAllocation(destinations: Destination[], index: number) {
  const otherTotal = destinations.reduce((total, item, itemIndex) => total + (itemIndex !== index && item.active && !item.archived ? item.allocationBps : 0), 0)
  return destinations.map((item, itemIndex) => itemIndex === index ? { ...item, allocationBps: Math.max(0, Math.min(10000, 10000 - otherTotal)) } : item)
}

export function distributeAllocationsEvenly(destinations: Destination[]) {
  const activeIndexes = destinations.flatMap((item, index) => item.active && !item.archived ? [index] : [])
  if (!activeIndexes.length) return destinations
  const basePercent = Math.floor(100 / activeIndexes.length)
  const extraPercentItems = 100 % activeIndexes.length
  const activeOrder = new Map(activeIndexes.map((index, order) => [index, order]))
  return destinations.map((item, index) => {
    const order = activeOrder.get(index)
    return order === undefined ? item : { ...item, allocationBps: (basePercent + (order < extraPercentItems ? 1 : 0)) * 100 }
  })
}

function withSortOrder(destinations: Destination[]) {
  return destinations.map((item, sortOrder) => ({ ...item, sortOrder }))
}

export function reorderDestinations(destinations: Destination[], fromIndex: number, toIndex: number) {
  if (fromIndex === toIndex || fromIndex < 0 || toIndex < 0 || fromIndex >= destinations.length || toIndex >= destinations.length) return destinations
  const next = [...destinations]
  const [moved] = next.splice(fromIndex, 1)
  next.splice(toIndex, 0, moved)
  return withSortOrder(next)
}

export function sortDestinationsByAllocation(destinations: Destination[]) {
  return withSortOrder(destinations.map((item, index) => ({ item, index })).sort((left, right) => {
    const leftAllocation = left.item.active && !left.item.archived ? left.item.allocationBps : -1
    const rightAllocation = right.item.active && !right.item.archived ? right.item.allocationBps : -1
    return rightAllocation - leftAllocation || left.index - right.index
  }).map(({ item }) => item))
}

export function duplicateActiveTargetNames(destinations: Destination[]) {
  const seen = new Set<string>(); const duplicates = new Set<string>()
  for (const item of destinations.filter(item => item.active && !item.archived)) {
    const key = item.name.trim().toLocaleLowerCase()
    if (key && seen.has(key)) duplicates.add(key)
    seen.add(key)
  }
  return duplicates
}
