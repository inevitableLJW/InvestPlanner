import { expect, it } from 'vitest'
import styles from './styles.css?raw'

it('keeps open and limited purchases in the same green status style', () => {
  const rule = styles.match(/\.purchase\.open\s*,\s*\.purchase\.limited\s*\{([^}]+)\}/)?.[1]
  expect(rule).toContain('color: #15754d')
  expect(rule).toContain('background: #e2f6ed')
})

it('centers the ETF tracking-error heading and values in the same column', () => {
  const rule = styles.match(/\.etf-table th:nth-child\(6\)\s*,\s*\.etf-table td:nth-child\(6\)\s*\{([^}]+)\}/)?.[1]
  expect(rule).toContain('text-align: center')
})
