import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { AuthProvider } from './auth/AuthContext'

const json = (value: unknown, status = 200) => Promise.resolve(new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json' } }))
afterEach(() => { vi.restoreAllMocks(); window.localStorage.clear() })

function renderApp(path: string) {
  return render(<MemoryRouter initialEntries={[path]}><AuthProvider><App /></AuthProvider></MemoryRouter>)
}

describe('authentication routes', () => {
  it('protects plan routes when the session is missing', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => json({ code: 'unauthorized', message: '请先登录' }, 401))
    renderApp('/plans')
    expect(await screen.findByRole('heading', { name: '欢迎回来' })).toBeInTheDocument()
  })

  it('registers and safely opens the plan list', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockImplementationOnce(() => json({ code: 'unauthorized', message: '请先登录' }, 401))
      .mockImplementationOnce(() => json({ id: 'u1', username: 'new-user' }, 201))
      .mockImplementationOnce(() => json({ items: [] }))
    renderApp('/register')
    await userEvent.type(await screen.findByLabelText('用户名'), 'new-user')
    await userEvent.type(screen.getByLabelText('密码'), 'securepass1')
    await userEvent.click(screen.getByRole('button', { name: '注册并开始' }))
    expect(await screen.findByRole('heading', { name: /让每月结余/ })).toBeInTheDocument()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3))
  })
})

describe('plan-first onboarding', () => {
  it('shows the explicit create action and explains common investment targets', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockImplementationOnce(() => json({ id: 'u1', username: 'user' }))
      .mockImplementationOnce(() => json({ items: [] }))
    renderApp('/plans')
    expect(await screen.findByRole('button', { name: '创建定投计划' })).toBeInTheDocument()
    expect(screen.getByText(/可直接选择现金/)).toHaveTextContent('自定义标的')
  })
})
describe('investment target settings', () => {
  it('supports common targets, progressive custom entry, even allocation and removal', async () => {
    const plan = {
      id: 'p1', name: '长期计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 100, version: 1,
      destinations: ['现金', '债券类基金', '纳斯达克100指数（QDII）', '标普500指数（QDII）'].map((name, index) => ({
        id: `d${index}`, name, active: true, archived: false, sortOrder: index, allocationBps: 0, version: 1,
      })),
    }
    vi.spyOn(globalThis, 'fetch')
      .mockImplementationOnce(() => json({ id: 'u1', username: 'user' }))
      .mockImplementationOnce(() => json(plan))
    renderApp('/plans/p1/settings')
    expect(await screen.findByLabelText('投资标的 1')).toHaveValue('现金')
    expect(screen.getByText(/还剩 100% 待分配/)).toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('投资标的 1'), '__custom__')
    await userEvent.type(screen.getByLabelText('自定义标的名称 1'), '黄金 ETF')
    expect(screen.getByLabelText('自定义标的名称 1')).toHaveValue('黄金 ETF')

    await userEvent.click(screen.getByRole('button', { name: '智能均分到 100%' }))
    expect(screen.getByLabelText('黄金 ETF分配比例')).toHaveValue('25')
    expect(screen.getByText('配置完整，可以保存启用')).toBeInTheDocument()

    await userEvent.click(screen.getAllByRole('button', { name: '移除' })[0])
    expect(screen.queryByDisplayValue('黄金 ETF')).not.toBeInTheDocument()
  })

  it('sorts and drags targets, then confirms a save by returning to overview', async () => {
    const plan = {
      id: 'p1', name: '长期计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 100, version: 1,
      destinations: [
        { id: 'd1', name: '现金', allocationBps: 2000 },
        { id: 'd2', name: '债券类基金', allocationBps: 5000 },
        { id: 'd3', name: '美股', allocationBps: 3000 },
      ].map((item, index) => ({ ...item, active: true, archived: false, sortOrder: index, version: 1 })),
    }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/plans/p1') && init?.method === 'PUT') return json(plan)
      if (url.endsWith('/plans/p1')) return json(plan)
      return json({ items: [] })
    })
    renderApp('/plans/p1/settings')
    await screen.findByLabelText('投资标的 1')
    await userEvent.click(screen.getByRole('button', { name: '按比例从高到低' }))
    expect(screen.getAllByLabelText(/投资标的 \d+/).map(element => (element as HTMLSelectElement).value)).toEqual(['债券类基金', '美股', '现金'])

    const transfer = { effectAllowed: '', setData: vi.fn(), getData: vi.fn(() => '0') }
    const handles = screen.getAllByRole('button', { name: /拖动排序/ })
    fireEvent.dragStart(handles[0], { dataTransfer: transfer })
    fireEvent.drop(screen.getByLabelText('现金分配比例').closest('fieldset')!, { dataTransfer: transfer })
    expect(screen.getAllByLabelText(/投资标的 \d+/).map(element => (element as HTMLSelectElement).value)).toEqual(['美股', '现金', '债券类基金'])

    await userEvent.click(screen.getByRole('button', { name: '保存计划设置' }))
    expect(await screen.findByRole('dialog', { name: '设置已保存' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: '确认并返回概览' }))
    expect(await screen.findByRole('heading', { name: '投资标的配置' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/plans/p1'), expect.objectContaining({ method: 'PUT' }))
  })

  it('uses stable tab paths and remembers the selected skin', async () => {
    const plan = {
      id: 'p1', name: '长期计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 100, version: 1, destinations: [],
    }
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/plans/p1/stats')) return json({ Destinations: [], RecommendedTotalCents: 0, ActualTotalCents: 0, CompletionRate: null })
      if (url.endsWith('/plans/p1')) return json(plan)
      return json({ items: [] })
    })
    renderApp('/plans/p1/settings')
    const settingsTab = await screen.findByRole('link', { name: '计划设置' })
    expect(settingsTab).toHaveAttribute('href', '/plans/p1/settings')
    expect(screen.getByRole('link', { name: '投入统计' })).toHaveAttribute('href', '/plans/p1/stats')
    await userEvent.click(screen.getByRole('link', { name: '投入统计' }))
    expect(await screen.findByText('暂无实际投入数据')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /选择界面皮肤/ }))
    const violetOption = await screen.findByRole('option', { name: '暮光紫' })
    expect(violetOption.querySelector('.skin-swatch')).toBeInTheDocument()
    await userEvent.click(violetOption)
    expect(document.querySelector('.app-shell')).toHaveAttribute('data-skin', 'violet')
    expect(window.localStorage.getItem('invest-plan-skin')).toBe('violet')
    await userEvent.click(screen.getByRole('button', { name: '☾ 深色' }))
    expect(document.querySelector('.app-shell')).toHaveAttribute('data-mode', 'dark')
    expect(document.documentElement).toHaveAttribute('data-display-mode', 'dark')
    expect(window.localStorage.getItem('invest-plan-display-mode')).toBe('dark')
  })
})
  it('logs in, restores access, and logs out', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockImplementationOnce(() => json({ error: { code: 'unauthorized', message: '请先登录' } }, 401))
      .mockImplementationOnce(() => json({ id: 'u1', username: 'user' }))
      .mockImplementationOnce(() => json({ items: [] }))
      .mockImplementationOnce(() => Promise.resolve(new Response(null, { status: 204 })))
    renderApp('/login')
    await userEvent.type(await screen.findByLabelText('用户名'), 'user')
    await userEvent.type(screen.getByLabelText('密码'), 'securepass1')
    await userEvent.click(screen.getByRole('button', { name: '登录' }))
    expect(await screen.findByRole('heading', { name: /让每月结余/ })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: '退出' }))
    expect(await screen.findByRole('heading', { name: '欢迎回来' })).toBeInTheDocument()
  })
