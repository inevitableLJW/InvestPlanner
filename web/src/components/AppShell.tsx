import { useEffect, useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

const SKINS = [
  { value: 'ocean', label: '海洋蓝', color: '#286bd6' },
  { value: 'violet', label: '暮光紫', color: '#7354ce' },
  { value: 'teal', label: '青绿色', color: '#118b91' },
] as const

type Skin = (typeof SKINS)[number]['value']
type DisplayMode = 'light' | 'dark'

function initialSkin(): Skin {
  if (typeof window === 'undefined') return 'ocean'
  const saved = window.localStorage.getItem('invest-plan-skin')
  return SKINS.some(skin => skin.value === saved) ? saved as Skin : 'ocean'
}

function initialDisplayMode(): DisplayMode {
  if (typeof window === 'undefined') return 'light'
  const saved = window.localStorage.getItem('invest-plan-display-mode')
  return saved === 'dark' || saved === 'light' ? saved : 'light'
}

export function AppShell() {
  const { user, logout } = useAuth()
  const [skin, setSkin] = useState<Skin>(initialSkin)
  const [displayMode, setDisplayMode] = useState<DisplayMode>(initialDisplayMode)
  const [skinMenuOpen, setSkinMenuOpen] = useState(false)
  const selectedSkin = SKINS.find(option => option.value === skin) ?? SKINS[0]
  useEffect(() => { window.localStorage.setItem('invest-plan-skin', skin) }, [skin])
  useEffect(() => {
    window.localStorage.setItem('invest-plan-display-mode', displayMode)
    document.documentElement.dataset.displayMode = displayMode
  }, [displayMode])
  return <div className="app-shell" data-skin={skin} data-mode={displayMode}>
    <header className="topbar">
      <NavLink className="brand" to="/plans"><span className="brand-mark"><img src="/invest-plan-logo.png" alt="" aria-hidden="true" /></span><span>Invest<span>Plan</span></span></NavLink>
      <nav aria-label="主导航"><NavLink to="/plans">我的计划</NavLink></nav>
      <div className="account"><div className="skin-picker"><span className="appearance-label">皮肤</span><button type="button" className="skin-trigger" aria-label={`选择界面皮肤，当前${selectedSkin.label}`} aria-haspopup="listbox" aria-expanded={skinMenuOpen} onClick={() => setSkinMenuOpen(open => !open)}><i className="skin-swatch" style={{ backgroundColor: selectedSkin.color }} /><span>{selectedSkin.label}</span><b>⌄</b></button>{skinMenuOpen && <div className="skin-menu" role="listbox" aria-label="界面皮肤">{SKINS.map(option => <button type="button" role="option" aria-selected={option.value === skin} key={option.value} onClick={() => { setSkin(option.value); setSkinMenuOpen(false) }}><i className="skin-swatch" style={{ backgroundColor: option.color }} /><span>{option.label}</span>{option.value === skin && <b aria-hidden="true">✓</b>}</button>)}</div>}</div><div className="mode-toggle" role="group" aria-label="背景模式"><button type="button" aria-pressed={displayMode === 'light'} onClick={() => setDisplayMode('light')}>☀ 浅色</button><button type="button" aria-pressed={displayMode === 'dark'} onClick={() => setDisplayMode('dark')}>☾ 深色</button></div><span className="avatar">{user?.username?.slice(0, 1).toUpperCase()}</span><span>{user?.username}</span><button className="button ghost" onClick={() => void logout()}>退出</button></div>
    </header>
    <main className="page"><Outlet /></main>
  </div>
}
