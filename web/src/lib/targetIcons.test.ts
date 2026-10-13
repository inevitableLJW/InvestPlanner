import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { COMMON_INVESTMENT_TARGETS } from './investmentTargets'
import { getTargetIconKey, targetIconPath } from './targetIcons'

describe('investment target icons', () => {
  it('covers every selectable preset with a distinct asset', () => {
    const keys = COMMON_INVESTMENT_TARGETS.map(getTargetIconKey)
    expect(keys).not.toContain('custom')
    expect(new Set(keys).size).toBe(COMMON_INVESTMENT_TARGETS.length)
    for (const key of [...keys, 'custom'] as const) {
      expect(targetIconPath(key)).toMatch(/^\/target-icons\/[a-z0-9-]+\.png$/)
      const png = readFileSync(resolve(`public${targetIconPath(key)}`))
      expect(png.subarray(0, 8).toString('hex')).toBe('89504e470d0a1a0a')
      expect(png.readUInt32BE(16)).toBe(128)
      expect(png.readUInt32BE(20)).toBe(128)
    }
  })

  it('preserves legacy names and separates the index investment routes', () => {
    expect(getTargetIconKey('  纳指（QDII）  ')).toBe(getTargetIconKey('纳斯达克100指数（QDII）'))
    expect(getTargetIconKey('标普（美股）')).toBe(getTargetIconKey('标普500指数（美股）'))
    expect(getTargetIconKey('纳指（QDII）')).not.toBe(getTargetIconKey('纳指（美股）'))
    expect(getTargetIconKey('标普（QDII）')).not.toBe(getTargetIconKey('标普（美股）'))
  })

  it('uses a neutral icon for arbitrary custom names, even those containing preset words', () => {
    for (const name of ['', '黄金 ETF', '我的美股组合', '港股通自选', 'constructor', '__proto__']) {
      expect(getTargetIconKey(name)).toBe('custom')
    }
  })
})
