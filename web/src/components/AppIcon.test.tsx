import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { getExpenseIconMeta } from '../lib/expenseIcons'
import { AppIcon } from './AppIcon'

describe('expense source icon mapping', () => {
  it('maps common apps and categories to recognizable icons', () => {
    expect(getExpenseIconMeta('微信').kind).toBe('wechat')
    expect(getExpenseIconMeta('支付宝').kind).toBe('alipay')
    expect(getExpenseIconMeta('美团外卖').kind).toBe('meituan')
    expect(getExpenseIconMeta('京东商城').kind).toBe('jd')
    expect(getExpenseIconMeta('银行卡').kind).toBe('bank')
    expect(getExpenseIconMeta('交通').kind).toBe('transport')
  })

  it('renders a stable wallet fallback for custom sources', () => {
    render(<AppIcon name="线下商店" />)
    expect(screen.getByRole('img', { name: '自定义来源图标' })).toHaveAttribute('data-icon', 'wallet')
  })
})
