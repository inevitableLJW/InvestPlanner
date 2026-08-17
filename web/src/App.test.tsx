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

  it('shows and hides the password without submitting the login form', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(() => json({ code: 'unauthorized', message: '请先登录' }, 401))
    renderApp('/login')

    const password = await screen.findByLabelText('密码')
    expect(password).toHaveAttribute('type', 'password')

    const showPasswordButton = screen.getByRole('button', { name: '显示密码' })
    expect(showPasswordButton.querySelector('svg')).toBeInTheDocument()
    expect(showPasswordButton).toHaveTextContent('')
    await userEvent.click(showPasswordButton)
    expect(password).toHaveAttribute('type', 'text')
    expect(screen.getByRole('button', { name: '隐藏密码' })).toHaveAttribute('aria-pressed', 'true')

    await userEvent.click(screen.getByRole('button', { name: '隐藏密码' }))
    expect(password).toHaveAttribute('type', 'password')
    expect(fetchMock).toHaveBeenCalledTimes(1)
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
    expect(screen.getByText('--')).toBeInTheDocument()
    expect(screen.getByText(/可直接选择现金/)).toHaveTextContent('自定义标的')
  })
})
describe('investment target settings', () => {
  it('supports common targets, progressive custom entry, even allocation and removal', async () => {
    const plan = {
      id: 'p1', name: '长期计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: true,
      destinations: ['现金', '债券类基金', '纳斯达克100指数（QDII）', '标普500指数（QDII）'].map((name, index) => ({
        id: `d${index}`, name, active: true, archived: false, sortOrder: index, allocationBps: 0, version: 1,
      })),
    }
    vi.spyOn(globalThis, 'fetch')
      .mockImplementationOnce(() => json({ id: 'u1', username: 'user' }))
      .mockImplementationOnce(() => json(plan))
    renderApp('/plans/p1/settings')
    expect(await screen.findByLabelText('投资标的 1')).toHaveValue('现金')
    expect(screen.getByLabelText(/向下取整单位/)).toHaveValue(100)
    expect(screen.getByText(/还剩 100% 待分配/)).toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('投资标的 1'), '__custom__')
    await userEvent.type(screen.getByLabelText('自定义标的名称 1'), '黄金 ETF')
    expect(screen.getByLabelText('自定义标的名称 1')).toHaveValue('黄金 ETF')

    await userEvent.click(screen.getByRole('button', { name: '智能均分到 100%' }))
    expect(screen.getByLabelText('黄金 ETF分配比例')).toHaveValue('25')
    expect(screen.getByText('配置完整，可以发布')).toBeInTheDocument()

    await userEvent.click(screen.getAllByRole('button', { name: '移除' })[0])
    expect(screen.queryByDisplayValue('黄金 ETF')).not.toBeInTheDocument()
  })

  it('sorts and drags targets, then confirms a save by returning to overview', async () => {
    const plan = {
      id: 'p1', name: '长期计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: true,
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

    await userEvent.click(screen.getByRole('button', { name: '保存草稿' }))
    expect(await screen.findByRole('dialog', { name: '草稿已保存' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: '确认并返回概览' }))
    expect(await screen.findByRole('heading', { name: '投资标的配置' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/plans/p1'), expect.objectContaining({ method: 'PUT' }))
    const updateCall = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(updateCall?.[1]?.body))).toMatchObject({ action: 'save_draft' })
  })

  it('saves incomplete settings as a draft and gates publishing', async () => {
    const plan = {
      id: 'p1', name: '未完成计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: true,
      destinations: [{ id: 'd1', name: '现金', active: true, archived: false, sortOrder: 0, allocationBps: 0, version: 1 }],
    }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/plans/p1') && init?.method === 'PUT') return json({ ...plan, version: 2 })
      if (url.endsWith('/plans/p1')) return json(plan)
      return json({ items: [] })
    })
    renderApp('/plans/p1/settings')
    expect(await screen.findByRole('button', { name: '发布计划' })).toBeDisabled()
    await userEvent.click(screen.getByRole('button', { name: '保存草稿' }))
    expect(await screen.findByRole('dialog', { name: '草稿已保存' })).toHaveTextContent('计划尚未开始运行')
    const updateCall = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(updateCall?.[1]?.body))).toMatchObject({ action: 'save_draft' })
  })

  it('publishes ready settings with distinct feedback and hides draft deletion', async () => {
    const plan = {
      id: 'p1', name: '发布计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: true,
      destinations: [{ id: 'd1', name: '现金', active: true, archived: false, sortOrder: 0, allocationBps: 10000, version: 1 }],
    }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/plans/p1') && init?.method === 'PUT') return json({ ...plan, status: 'active', version: 2, deletable: false })
      if (url.endsWith('/plans/p1')) return json(plan)
      return json({ items: [] })
    })
    renderApp('/plans/p1/settings')
    await userEvent.click(await screen.findByRole('button', { name: '发布计划' }))
    expect(await screen.findByRole('dialog', { name: '计划已发布' })).toHaveTextContent('计划已经开始运行')
    const updateCall = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(updateCall?.[1]?.body))).toMatchObject({ action: 'publish' })
    expect(screen.queryByRole('button', { name: '删除草稿计划' })).not.toBeInTheDocument()
  })

  it('warns before saving a running plan back to draft', async () => {
    const plan = {
      id: 'p1', name: '运行中计划', status: 'active', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: false,
      destinations: [{ id: 'd1', name: '现金', active: true, archived: false, sortOrder: 0, allocationBps: 10000, version: 1 }],
    }
    const confirmMock = vi.spyOn(window, 'confirm').mockReturnValue(true)
    vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/plans/p1') && init?.method === 'PUT') return json({ ...plan, status: 'draft', version: 2, deletable: true })
      if (url.endsWith('/plans/p1')) return json(plan)
      return json({ items: [] })
    })
    renderApp('/plans/p1/settings')
    await userEvent.click(await screen.findByRole('button', { name: '保存草稿' }))
    expect(confirmMock).toHaveBeenCalledWith(expect.stringContaining('计划将暂停运行'))
    expect(await screen.findByRole('dialog', { name: '草稿已保存' })).toBeInTheDocument()
  })

  it('confirms and permanently deletes an eligible draft from the plan overview without duplicate submission', async () => {
    const plan = {
      id: 'p1', name: '待删除计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: true, destinations: [],
    }
    let resolveDelete!: (response: Response) => void
    const pendingDelete = new Promise<Response>(resolve => { resolveDelete = resolve })
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/plans/p1/draft?version=1') && init?.method === 'DELETE') return pendingDelete
      if (url.endsWith('/plans')) return json({ items: [plan] })
      return json({ items: [] })
    })
    renderApp('/plans')
    await userEvent.click(await screen.findByRole('button', { name: '删除草稿' }))
    expect(screen.getByRole('dialog', { name: '永久删除“待删除计划”？' })).toHaveTextContent('无法恢复')
    await userEvent.click(screen.getByRole('button', { name: '取消' }))
    expect(screen.queryByRole('dialog', { name: '永久删除“待删除计划”？' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: '删除草稿' }))
    await userEvent.click(screen.getByRole('button', { name: '确认永久删除' }))
    expect(screen.getByRole('button', { name: '删除中…' })).toBeDisabled()
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === 'DELETE')).toHaveLength(1)
    resolveDelete(new Response(null, { status: 204 }))
    await waitFor(() => expect(screen.queryByRole('heading', { name: '待删除计划' })).not.toBeInTheDocument())
  })

  it('keeps the draft visible when deletion eligibility is stale', async () => {
    const plan = {
      id: 'p1', name: '状态已变化', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: true, destinations: [],
    }
    vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/plans/p1/draft?version=1') && init?.method === 'DELETE') return json({ error: { code: 'conflict', message: '数据已被修改，请刷新后重试' } }, 409)
      if (url.endsWith('/plans')) return json({ items: [plan] })
      return json({ items: [] })
    })
    renderApp('/plans')
    await userEvent.click(await screen.findByRole('button', { name: '删除草稿' }))
    await userEvent.click(screen.getByRole('button', { name: '确认永久删除' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('计划未删除')
    expect(screen.getByRole('heading', { name: '状态已变化' })).toBeInTheDocument()
  })

  it('uses stable tab paths and remembers the selected skin', async () => {
    const plan = {
      id: 'p1', name: '长期计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: true, destinations: [],
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

describe('monthly allocation explanation', () => {
  it('explains that cash receives per-target rounding remainder', async () => {
    const plan = {
      id: 'p1', name: '现金尾差计划', status: 'active', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 10000, version: 2, deletable: false,
      destinations: [
        { id: 'd1', name: 'A股', active: true, archived: false, sortOrder: 0, allocationBps: 7000, version: 1 },
        { id: 'd2', name: '现金', active: true, archived: false, sortOrder: 1, allocationBps: 3000, version: 1 },
      ],
    }
    const record = {
      id: 'm1', planId: 'p1', month: '2026-08', incomeCents: 100000, expenseTotalCents: 0,
      reserveCents: 0, surplusCents: 100000, investableBaseCents: 100000, contributionBps: 10000,
      recommendedTotalCents: 100000, roundingUnitCents: 10000, actualTotalCents: 0,
      status: 'not_started', note: '', version: 1, expenses: [],
      allocations: [
        { id: 'a1', destinationId: 'd1', name: 'A股', sortOrder: 0, allocationBps: 7000, recommendedCents: 70000, actualCents: 0, differenceCents: -70000 },
        { id: 'a2', destinationId: 'd2', name: '现金', sortOrder: 1, allocationBps: 3000, recommendedCents: 30000, actualCents: 0, differenceCents: -30000 },
      ],
    }
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/plans/p1/months/2026-08')) return json(record)
      if (url.endsWith('/expense-sources')) return json({ items: [] })
      if (url.endsWith('/plans/p1')) return json(plan)
      return json({ items: [] })
    })
    renderApp('/plans/p1/month/2026-08')
    expect(await screen.findByText(/尾差（包括零头）全部归入现金/)).toBeInTheDocument()
  })
})

describe('monthly income expression', () => {
  it('evaluates simple income addition and subtraction while keeping the investable amount exact', async () => {
    const plan = {
      id: 'p1', name: '月度计划', status: 'active', defaultContributionBps: 10000,
      reserveCents: 0, roundingUnitCents: 10000, version: 1, deletable: false, destinations: [],
    }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.endsWith('/auth/me')) return json({ id: 'u1', username: 'user' })
      if (url.endsWith('/expense-sources')) return json({ items: [{ id: 'e1', name: '支付宝', active: true, sortOrder: 0 }] })
      if (url.includes('/months/')) return json({ error: { code: 'not_found', message: '未找到' } }, 404)
      if (url.endsWith('/plans/p1')) return json(plan)
      return json({ items: [] })
    })
    renderApp('/plans/p1/month/2026-08')
    await userEvent.type(await screen.findByLabelText('上月总收入'), '20000+500-100')
    expect(screen.getByText('当前合计 ¥20,400.00 · 支持 +、- 运算')).toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('支付宝支出金额'), '3637')
    expect(await screen.findByText('¥16,763.00')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: '生成本月投资建议 →' }))
    await waitFor(() => expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'PUT')).toBe(true))
    const updateCall = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(updateCall?.[1]?.body))).toMatchObject({ incomeCents: 2040000 })
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
