export const yuan = (cents: number) => new Intl.NumberFormat('zh-CN', {
  style: 'currency', currency: 'CNY', minimumFractionDigits: 2,
}).format(cents / 100)

export const fromYuan = (value: string) => Math.round((Number(value) || 0) * 100)
export const toYuan = (cents: number) => cents ? String(cents / 100) : ''
export const percent = (bps: number) => `${(bps / 100).toFixed(2).replace(/\.00$/, '')}%`
