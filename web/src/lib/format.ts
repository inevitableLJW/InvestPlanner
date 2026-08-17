export const yuan = (cents: number) => new Intl.NumberFormat('zh-CN', {
  style: 'currency', currency: 'CNY', minimumFractionDigits: 2,
}).format(cents / 100)

export const fromYuan = (value: string) => Math.round((Number(value) || 0) * 100)
export const toYuan = (cents: number) => cents ? String(cents / 100) : ''
export const percent = (bps: number) => `${(bps / 100).toFixed(2).replace(/\.00$/, '')}%`

export function parseYuanExpression(value: string): number | null {
  const expression = value.replace(/\s/g, '').replaceAll('＋', '+').replaceAll('－', '-')
  if (!expression) return 0
  const terms = expression.match(/[+-]?\d+(?:\.\d{1,2})?/g)
  if (!terms || terms.join('') !== expression || terms[0].startsWith('-')) return null
  const cents = terms.reduce((total, term) => total + Math.round(Number(term) * 100), 0)
  return Number.isSafeInteger(cents) && cents >= 0 ? cents : null
}
