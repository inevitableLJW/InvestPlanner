import { normalizeInvestmentTargetName } from './investmentTargets'

export const TARGET_ICON_KEYS = {
  '现金': 'cash',
  '债券类基金': 'bond-fund',
  '纳斯达克100指数（QDII）': 'nasdaq-100-qdii',
  '标普500指数（QDII）': 'sp500-qdii',
  '红利低波': 'dividend-low-vol',
  'A股主动基金': 'a-share-active-fund',
  '沪深300指数基金': 'csi-300-fund',
  '中证500指数基金': 'csi-500-fund',
  'A股': 'a-share',
  '港股': 'hk-stock',
  '美股': 'us-stock',
  '纳斯达克100指数（美股）': 'nasdaq-100-us',
  '标普500指数（美股）': 'sp500-us',
} as const

export type TargetIconKey = (typeof TARGET_ICON_KEYS)[keyof typeof TARGET_ICON_KEYS] | 'custom'

export function getTargetIconKey(name: string): TargetIconKey {
  const normalized = normalizeInvestmentTargetName(name)
  return Object.hasOwn(TARGET_ICON_KEYS, normalized)
    ? TARGET_ICON_KEYS[normalized as keyof typeof TARGET_ICON_KEYS]
    : 'custom'
}

export function targetIconPath(key: TargetIconKey) {
  return `/target-icons/${key}.png`
}
