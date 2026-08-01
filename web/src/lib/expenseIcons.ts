export type ExpenseIconKind = 'wechat' | 'alipay' | 'meituan' | 'jd' | 'douyin' | 'bank' | 'transport' | 'other' | 'wallet'

export interface ExpenseIconMeta { kind: ExpenseIconKind; label: string; color: string; background: string }

const SOURCE_ICONS: Array<[RegExp, ExpenseIconMeta]> = [
  [/微信|wechat/i, { kind: 'wechat', label: '微信', color: '#fff', background: '#18b45b' }],
  [/支付宝|alipay/i, { kind: 'alipay', label: '支付宝', color: '#fff', background: '#1677ff' }],
  [/美团|meituan/i, { kind: 'meituan', label: '美团', color: '#20211e', background: '#ffd100' }],
  [/京东|京东商城|jd\.com|jingdong/i, { kind: 'jd', label: '京东', color: '#fff', background: '#e2231a' }],
  [/抖音|douyin|tiktok/i, { kind: 'douyin', label: '抖音', color: '#fff', background: '#17181c' }],
  [/银行|信用卡|储蓄卡|bank/i, { kind: 'bank', label: '银行卡', color: '#fff', background: '#7857d8' }],
  [/交通|地铁|公交|打车|transport/i, { kind: 'transport', label: '交通', color: '#fff', background: '#ef704a' }],
  [/其他|其它|other/i, { kind: 'other', label: '其他', color: '#3e4c45', background: '#e8eeea' }],
]

const FALLBACK_ICON: ExpenseIconMeta = { kind: 'wallet', label: '自定义来源', color: '#fff', background: '#2b70d6' }

export function getExpenseIconMeta(name: string): ExpenseIconMeta {
  return SOURCE_ICONS.find(([pattern]) => pattern.test(name.trim()))?.[1] ?? FALLBACK_ICON
}
