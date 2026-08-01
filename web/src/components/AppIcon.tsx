import type { CSSProperties, ReactNode } from 'react'
import { siAlipay, siMeituan, siWechat } from 'simple-icons'
import { getExpenseIconMeta, type ExpenseIconKind } from '../lib/expenseIcons'

function Svg({ children }: { children: ReactNode }) {
  return <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">{children}</svg>
}

function Glyph({ kind }: { kind: ExpenseIconKind }) {
  if (kind === 'wechat') return <Svg><path d={siWechat.path} fill="currentColor" /></Svg>
  if (kind === 'alipay') return <Svg><path d={siAlipay.path} fill="currentColor" /></Svg>
  if (kind === 'meituan') return <Svg><path d={siMeituan.path} fill="currentColor" /></Svg>
  if (kind === 'jd') return <span className="app-icon-word jd-word">JD</span>
  if (kind === 'douyin') return <Svg><path d="M14.2 4.2c.4 2.2 1.7 3.5 4 3.9v3.1a8.6 8.6 0 0 1-4-1.2v5.3a5.2 5.2 0 1 1-4.5-5.1v3.2a2.1 2.1 0 1 0 1.4 2V4.2h3.1Z" fill="currentColor"/></Svg>
  if (kind === 'bank') return <Svg><path d="m4 9 8-4 8 4v2H4V9Zm2 3h2v5H6v-5Zm5 0h2v5h-2v-5Zm5 0h2v5h-2v-5ZM4 18h16v2H4v-2Z" fill="currentColor"/></Svg>
  if (kind === 'transport') return <Svg><path d="M7 4h10c2 0 3 1.7 3 4v7a2 2 0 0 1-2 2h-.3l1.3 2h-2.4l-1.3-2H8.7l-1.3 2H5l1.3-2H6a2 2 0 0 1-2-2V8c0-2.3 1-4 3-4Zm0 2c-.6 0-1 .7-1 2v2h12V8c0-1.3-.4-2-1-2H7Zm.5 8.6a1.3 1.3 0 1 0 0-2.6 1.3 1.3 0 0 0 0 2.6Zm9 0a1.3 1.3 0 1 0 0-2.6 1.3 1.3 0 0 0 0 2.6Z" fill="currentColor"/></Svg>
  if (kind === 'other') return <Svg><circle cx="6" cy="12" r="1.6" fill="currentColor"/><circle cx="12" cy="12" r="1.6" fill="currentColor"/><circle cx="18" cy="12" r="1.6" fill="currentColor"/></Svg>
  return <Svg><path d="M5 6.5h12a2 2 0 0 1 2 2V10h1a1 1 0 0 1 1 1v5a3 3 0 0 1-3 3H5a3 3 0 0 1-3-3V8.5a3 3 0 0 1 3-3Zm0 2h12v-1H5a1 1 0 0 0-1 1.2c.3-.1.6-.2 1-.2Zm10 3.5v3h4v-3h-4Zm1.2 1.5a.65.65 0 1 0 1.3 0 .65.65 0 0 0-1.3 0Z" fill="currentColor"/></Svg>
}

export function AppIcon({ name, size = 'md' }: { name: string; size?: 'sm' | 'md' | 'lg' }) {
  const meta = getExpenseIconMeta(name)
  const style = { '--icon-bg': meta.background, background: meta.background, color: meta.color } as CSSProperties
  return <span className={`app-icon ${size}`} style={style} role="img" aria-label={`${meta.label}图标`} data-icon={meta.kind}><Glyph kind={meta.kind} /></span>
}

export function TargetIcon({ name }: { name: string }) {
  const safeName = name.trim()
  const text = safeName === '现金' ? '¥' : safeName.includes('债券') ? '债' : safeName.includes('纳斯达克') || safeName.includes('纳指') ? 'N' : safeName.includes('标普') ? 'S' : safeName.includes('港股') ? 'H' : safeName.includes('美股') ? 'U' : safeName.includes('A股') || safeName.includes('沪深') ? 'A' : safeName.includes('红利') ? '红' : '投'
  return <span className="target-icon" aria-hidden="true">{text}</span>
}
