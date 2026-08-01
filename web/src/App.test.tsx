import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { AuthProvider } from './auth/AuthContext'

const json = (value: unknown, status = 200) => Promise.resolve(new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json' } }))
afterEach(() => vi.restoreAllMocks())

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
    expect(await screen.findByRole('heading', { name: '我的定投计划' })).toBeInTheDocument()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3))
  })
})

describe('plan-first onboarding', () => {
  it('shows the explicit create action and explains optional defaults', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockImplementationOnce(() => json({ id: 'u1', username: 'user' }))
      .mockImplementationOnce(() => json({ items: [] }))
    renderApp('/plans')
    expect(await screen.findByRole('button', { name: '创建定投计划' })).toBeInTheDocument()
    expect(screen.getByText(/四个可编辑目的地/)).toHaveTextContent('均非必选')
  })
})
describe('destination settings', () => {
  it('renders optional defaults and supports custom entries and removal', async () => {
    const plan = {
      id: 'p1', name: '长期计划', status: 'draft', defaultContributionBps: 8000,
      reserveCents: 0, roundingUnitCents: 100, version: 1,
      destinations: ['支付宝基金', 'A股', '港美股', '现金'].map((name, index) => ({
        id: `d${index}`, name, active: true, archived: false, sortOrder: index, allocationBps: 0, version: 1,
      })),
    }
    vi.spyOn(globalThis, 'fetch')
      .mockImplementationOnce(() => json({ id: 'u1', username: 'user' }))
      .mockImplementationOnce(() => json(plan))
    renderApp('/plans/p1/settings')
    expect(await screen.findByDisplayValue('支付宝基金')).toBeInTheDocument()
    expect(screen.getByDisplayValue('现金')).toBeInTheDocument()
    expect(screen.getByText('必须恰好为 100%')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: '添加目的地' }))
    expect(screen.getByDisplayValue('新目的地')).toBeInTheDocument()
    await userEvent.click(screen.getAllByRole('button', { name: '移除' })[0])
    expect(screen.queryByDisplayValue('支付宝基金')).not.toBeInTheDocument()
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
    expect(await screen.findByRole('heading', { name: '我的定投计划' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: '退出' }))
    expect(await screen.findByRole('heading', { name: '欢迎回来' })).toBeInTheDocument()
  })
