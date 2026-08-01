import { describe, expect, it } from 'vitest'
import type { Destination } from '../types'
import {
  COMMON_INVESTMENT_TARGETS,
  CUSTOM_TARGET_VALUE,
  activeAllocationTotal,
  distributeAllocationsEvenly,
  duplicateActiveTargetNames,
  fillRemainingAllocation,
  investmentTargetSelectValue,
  normalizeInvestmentTargetName,
  reorderDestinations,
  sortDestinationsByAllocation,
} from './investmentTargets'

const targets = (...items: Array<Partial<Destination> & Pick<Destination, 'name'>>): Destination[] => items.map((item, index) => ({
  name: item.name, active: item.active ?? true, archived: item.archived ?? false,
  sortOrder: index, allocationBps: item.allocationBps ?? 0,
}))

describe('investment target presets', () => {
  it('contains common targets and keeps list-only names out of manual entry', () => {
    expect(COMMON_INVESTMENT_TARGETS).toContain('现金')
    expect(COMMON_INVESTMENT_TARGETS).toContain('纳斯达克100指数（QDII）')
    expect(COMMON_INVESTMENT_TARGETS).toContain('标普500指数（美股）')
    expect(COMMON_INVESTMENT_TARGETS).toContain('A股主动基金')
    expect(COMMON_INVESTMENT_TARGETS).toContain('沪深300指数基金')
    expect(COMMON_INVESTMENT_TARGETS).toContain('中证500指数基金')
    expect(COMMON_INVESTMENT_TARGETS).not.toContain('A股指数基金（如沪深300）')
    expect(investmentTargetSelectValue('现金')).toBe('现金')
    expect(normalizeInvestmentTargetName('纳指（QDII）')).toBe('纳斯达克100指数（QDII）')
    expect(normalizeInvestmentTargetName('A股指数基金（如沪深300）')).toBe('沪深300指数基金')
    expect(investmentTargetSelectValue('黄金 ETF')).toBe(CUSTOM_TARGET_VALUE)
  })
})

describe('linked allocation helpers', () => {
  it('distributes whole percentages and always totals 100%', () => {
    const result = distributeAllocationsEvenly(targets({ name: '现金' }, { name: '债券类基金' }, { name: '纳斯达克100指数（QDII）' }))
    expect(result.map(item => item.allocationBps)).toEqual([3400, 3300, 3300])
    expect(activeAllocationTotal(result)).toBe(10000)
  })

  it('fills the selected target with the exact remaining allocation', () => {
    const result = fillRemainingAllocation(targets(
      { name: '现金', allocationBps: 2000 },
      { name: '债券类基金', allocationBps: 3000 },
      { name: '美股', allocationBps: 0 },
    ), 2)
    expect(result[2].allocationBps).toBe(5000)
    expect(activeAllocationTotal(result)).toBe(10000)
  })

  it('detects duplicate enabled target names but ignores disabled ones', () => {
    expect(duplicateActiveTargetNames(targets({ name: '现金' }, { name: ' 现金 ' })).has('现金')).toBe(true)
    expect(duplicateActiveTargetNames(targets({ name: '现金' }, { name: '现金', active: false })).size).toBe(0)
  })

  it('sorts active targets by allocation and keeps equal allocations stable', () => {
    const result = sortDestinationsByAllocation(targets(
      { name: '现金', allocationBps: 2000 },
      { name: '债券类基金', allocationBps: 5000 },
      { name: '美股', allocationBps: 2000 },
      { name: '港股', active: false, allocationBps: 9000 },
    ))
    expect(result.map(item => item.name)).toEqual(['债券类基金', '现金', '美股', '港股'])
    expect(result.map(item => item.sortOrder)).toEqual([0, 1, 2, 3])
  })

  it('reorders a target and updates every sort order', () => {
    const result = reorderDestinations(targets({ name: '现金' }, { name: '债券类基金' }, { name: '美股' }), 2, 0)
    expect(result.map(item => item.name)).toEqual(['美股', '现金', '债券类基金'])
    expect(result.map(item => item.sortOrder)).toEqual([0, 1, 2])
  })
})
