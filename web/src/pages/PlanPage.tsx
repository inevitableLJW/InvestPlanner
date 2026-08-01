import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { Link, NavLink, Navigate, Route, Routes, useNavigate, useParams } from 'react-router-dom'
import { APIError, del, get, put } from '../lib/api'
import { fromYuan, percent, toYuan, yuan } from '../lib/format'
import type { Destination, Expense, ExpenseSource, MonthRecord, Plan, PlanStats } from '../types'

function usePlanData() {
  const { planId = '' } = useParams(); const [plan, setPlan] = useState<Plan | null>(null); const [error, setError] = useState('')
  useEffect(() => { void get<Plan>(`/plans/${planId}`).then(setPlan).catch(e => setError(e.message)) }, [planId])
  return { planId, plan, setPlan, error }
}

export function PlanPage() {
  const data = usePlanData()
  if (data.error) return <div className="alert" role="alert">{data.error} <Link to="/plans">返回计划列表</Link></div>
  if (!data.plan) return <div className="center" role="status">正在加载计划…</div>
  const { plan } = data
  return <>
    <section className="plan-title"><div><Link to="/plans">← 所有计划</Link><h1>{plan.name}</h1></div><span className={`badge ${plan.status}`}>{plan.status === 'active' ? '已启用' : plan.status === 'archived' ? '已归档' : '草稿'}</span></section>
    <nav className="tabs" aria-label="计划导航">
      <NavLink end to=".">概览</NavLink><NavLink to="settings">设置</NavLink><NavLink to="month">月度记录</NavLink><NavLink to="history">历史</NavLink><NavLink to="stats">统计</NavLink>
    </nav>
    <Routes>
      <Route index element={<Overview plan={plan} />} />
      <Route path="settings" element={<Settings plan={plan} onSaved={data.setPlan} />} />
      <Route path="month" element={<MonthEditor plan={plan} />} />
      <Route path="month/:month" element={<MonthEditor plan={plan} />} />
      <Route path="history" element={<History plan={plan} />} />
      <Route path="stats" element={<Stats plan={plan} />} />
      <Route path="*" element={<Navigate to="." replace />} />
    </Routes>
  </>
}

function Overview({ plan }: { plan: Plan }) {
  const active = plan.destinations.filter(d => d.active && !d.archived)
  const total = active.reduce((sum, d) => sum + d.allocationBps, 0)
  return <section className="stack">
    {plan.status !== 'active' && <div className="notice">当前计划不能新增月度记录。请在“设置”中保留至少一个目的地、使比例合计为 100%，然后启用计划。</div>}
    <div className="metric-grid"><Metric label="启用目的地" value={`${active.length} 个`} /><Metric label="分配比例" value={percent(total)} /><Metric label="预留现金" value={yuan(plan.reserveCents)} /><Metric label="默认投入率" value={percent(plan.defaultContributionBps)} /></div>
    <section className="panel"><h2>目的地配置</h2>{active.length ? active.map(d => <div className="allocation-row" key={d.id}><span>{d.name}</span><strong>{percent(d.allocationBps)}</strong></div>) : <p className="muted">尚未启用目的地。</p>}</section>
  </section>
}

function Settings({ plan, onSaved }: { plan: Plan; onSaved(plan: Plan): void }) {
  const [draft, setDraft] = useState(plan); const [message, setMessage] = useState(''); const [busy, setBusy] = useState(false)
  const activeTotal = draft.destinations.filter(d => d.active && !d.archived).reduce((s, d) => s + d.allocationBps, 0)
  const updateDestination = (index: number, update: Partial<Destination>) => setDraft({ ...draft, destinations: draft.destinations.map((d, i) => i === index ? { ...d, ...update } : d) })
  const addDestination = () => setDraft({ ...draft, destinations: [...draft.destinations, { name: '新目的地', active: true, archived: false, sortOrder: draft.destinations.length, allocationBps: 0 }] })
  const remove = (index: number) => setDraft({ ...draft, destinations: draft.destinations.filter((_, i) => i !== index).map((d, i) => ({ ...d, sortOrder: i })) })
  const move = (index: number, delta: number) => {
    const next = [...draft.destinations]; const target = index + delta
    if (target < 0 || target >= next.length) return
    ;[next[index], next[target]] = [next[target], next[index]]
    setDraft({ ...draft, destinations: next.map((d, i) => ({ ...d, sortOrder: i })) })
  }
  const save = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setMessage('')
    try {
      <div className="preset-row" aria-label="默认投入比例预设">{[50, 60, 80, 100].map(value => <button type="button" key={value} onClick={() => setDraft({ ...draft, defaultContributionBps: value * 100 })}>{value}%</button>)}</div>
      const result = await put<Plan>(`/plans/${plan.id}`, {
        name: draft.name, status: draft.status, defaultContributionBps: draft.defaultContributionBps,
        reserveCents: draft.reserveCents, roundingUnitCents: draft.roundingUnitCents, version: draft.version,
        destinations: draft.destinations.map(d => ({ id: d.id ?? '', name: d.name, active: d.active, allocationBps: d.allocationBps, sortOrder: d.sortOrder, version: d.version ?? 0 })),
      })
      setDraft(result); onSaved(result); setMessage('设置已保存')
    } catch (reason) { setMessage(reason instanceof APIError ? reason.message : '保存失败') }
    finally { setBusy(false) }
  }
  return <form className="stack" onSubmit={save}>
    {message && <div className={message === '设置已保存' ? 'success' : 'alert'} role="status">{message}</div>}
    <section className="panel form-grid"><h2>计划参数</h2>
      <label>计划名称<input required value={draft.name} onChange={e => setDraft({ ...draft, name: e.target.value })} /></label>
      <label>状态<select value={draft.status} onChange={e => setDraft({ ...draft, status: e.target.value as Plan['status'] })}><option value="draft">草稿</option><option value="active" disabled={activeTotal !== 10000}>启用</option><option value="archived">归档</option></select></label>
      <label>默认投入比例（%）<input type="number" min="0" max="100" step="0.01" value={draft.defaultContributionBps / 100} onChange={e => setDraft({ ...draft, defaultContributionBps: Math.round(Number(e.target.value) * 100) })} /></label>
      <label>预留金额（元）<input type="number" min="0" step="0.01" value={draft.reserveCents / 100} onChange={e => setDraft({ ...draft, reserveCents: fromYuan(e.target.value) })} /></label>
      <label>向下取整单位（元）<input type="number" min="0.01" step="0.01" value={draft.roundingUnitCents / 100} onChange={e => setDraft({ ...draft, roundingUnitCents: fromYuan(e.target.value) })} /></label>
    </section>
    <section className="panel"><div className="section-head"><div><h2>定投目的地</h2><p className="muted">四个默认项只是可编辑模板，不强制保留；也可以只使用一个 100% 目的地。</p></div><button type="button" className="button secondary" onClick={addDestination}>添加目的地</button></div>
      <div className={`total-bar ${activeTotal === 10000 ? 'valid' : 'invalid'}`}>启用项合计 <strong>{percent(activeTotal)}</strong><span>{activeTotal === 10000 ? '可启用' : '必须恰好为 100%'}</span></div>
      <div className="destination-list">{draft.destinations.map((d, index) => <fieldset className="destination" key={d.id ?? `new-${index}`} disabled={d.archived}>
        <label className="check"><input type="checkbox" checked={d.active} onChange={e => updateDestination(index, { active: e.target.checked })} />启用</label>
        <label className="grow">名称<input required value={d.name} onChange={e => updateDestination(index, { name: e.target.value })} /></label>
        <label>比例（%）<input type="number" min="0" max="100" step="0.01" value={d.allocationBps / 100} onChange={e => updateDestination(index, { allocationBps: Math.round(Number(e.target.value) * 100) })} /></label>
        <div className="inline-actions"><button type="button" aria-label="上移" onClick={() => move(index, -1)}>↑</button><button type="button" aria-label="下移" onClick={() => move(index, 1)}>↓</button><button type="button" onClick={() => remove(index)}>{d.archived ? '已归档' : '移除'}</button></div>
      </fieldset>)}</div>
    </section>
    <button className="button primary sticky-action" disabled={busy || (draft.status === 'active' && activeTotal !== 10000)}>{busy ? '保存中…' : '保存计划设置'}</button>
  </form>
}

function defaultMonth() { const date = new Date(); return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}` }
function blankExpenses(sources: ExpenseSource[]): Expense[] { return sources.filter(s => s.active).map(s => ({ sourceId: s.id, sourceName: s.name, amountCents: 0, sortOrder: s.sortOrder })) }

function MonthEditor({ plan }: { plan: Plan }) {
  const params = useParams(); const navigate = useNavigate(); const [month, setMonth] = useState(params.month ?? defaultMonth())
  const [income, setIncome] = useState(''); const [rate, setRate] = useState(plan.defaultContributionBps / 100); const [expenses, setExpenses] = useState<Expense[]>([])
  const [note, setNote] = useState(''); const [record, setRecord] = useState<MonthRecord | null>(null); const [message, setMessage] = useState(''); const [busy, setBusy] = useState(false)
  useEffect(() => {
	setRecord(null); setIncome(''); setRate(plan.defaultContributionBps / 100); setNote('')
    get<{ items: ExpenseSource[] }>('/expense-sources').then(r => setExpenses(blankExpenses(r.items)))
    get<MonthRecord>(`/plans/${plan.id}/months/${month}`).then(r => { setRecord(r); setIncome(toYuan(r.incomeCents)); setRate(r.contributionBps / 100); setExpenses(r.expenses); setNote(r.note) }).catch(e => { if (e.status !== 404) setMessage(e.message); else setRecord(null) })
  }, [plan.id, plan.defaultContributionBps, month])
  const expenseTotal = expenses.reduce((sum, e) => sum + e.amountCents, 0)
  const save = async (event: FormEvent) => {
    event.preventDefault()
    if (record && month < defaultMonth() && !window.confirm('这是历史月份。确认使用该月已保存的目的地快照重新计算？')) return
    setBusy(true); setMessage('')
    try {
      const result = await put<MonthRecord>(`/plans/${plan.id}/months/${month}`, {
        incomeCents: fromYuan(income), contributionBps: Math.round(rate * 100), expenses, note,
        ...(record ? { version: record.version } : {}),
      })
      setRecord(result); setMessage('计算与记录已保存'); navigate(`/plans/${plan.id}/month/${month}`, { replace: true })
    } catch (reason) { setMessage(reason instanceof APIError ? reason.message : '保存失败') }
    finally { setBusy(false) }
  }
  const copyLayout = async () => {
    try { const r = await get<{ items: Expense[] }>(`/plans/${plan.id}/months/previous-sources?before=${month}`); setExpenses(r.items.map(e => ({ ...e, amountCents: 0 }))); setMessage('已复制该计划上月的平台布局，金额保持为空') }
    catch (e) { setMessage(e instanceof APIError ? e.message : '没有可复制的上月布局') }
  }
  const updateActuals = async () => {
    if (!record) return
    try { const result = await put<MonthRecord>(`/plans/${plan.id}/months/${month}/actuals`, { version: record.version, items: record.allocations.map(a => ({ allocationId: a.id, actualCents: a.actualCents })) }); setRecord(result); setMessage('实际投入已更新') }
    catch (e) { setMessage(e instanceof APIError ? e.message : '更新失败') }
  }
  if (plan.status !== 'active') return <div className="empty panel"><h2>当前计划不能新增记录</h2><p>请先在设置中校验比例并启用计划。历史记录仍可在“历史”中查看。</p></div>
  return <div className="stack">
    {message && <div className={message.includes('失败') ? 'alert' : 'notice'} role="status">{message}</div>}
    <form className="panel stack" onSubmit={save}>
      <div className="section-head"><div><h2>月度收支</h2><p className="muted">支出按付款平台汇总；同一平台不要重复填写。不同计划可复用平台，但每份收支只属于当前计划。</p></div><label>月份<input type="month" value={month} onChange={e => setMonth(e.target.value)} /></label></div>
      <div className="preset-row" aria-label="本月投入比例预设">{[50, 60, 80, 100].map(value => <button type="button" key={value} onClick={() => setRate(value)}>{value}%</button>)}</div>
      <div className="form-grid three"><label>上月收入（元）<input type="number" min="0" step="0.01" value={income} onChange={e => setIncome(e.target.value)} /></label><label>本月投入比例（%）<input type="number" min="0" max="100" step="0.01" value={rate} onChange={e => setRate(Number(e.target.value))} /></label><div className="readout"><span>支出合计</span><strong>{yuan(expenseTotal)}</strong></div></div>
      <div className="section-head"><h3>按付款平台汇总支出</h3><button type="button" className="button secondary" onClick={() => void copyLayout()}>复制上月平台布局</button></div>
      <div className="expense-grid">{expenses.map((expense, index) => <label key={expense.sourceId || index}>{expense.sourceName}<input type="number" min="0" step="0.01" placeholder="0" value={toYuan(expense.amountCents)} onChange={e => setExpenses(expenses.map((x, i) => i === index ? { ...x, amountCents: fromYuan(e.target.value) } : x))} /></label>)}</div>
      <label>备注<textarea value={note} onChange={e => setNote(e.target.value)} /></label>
      <button className="button primary" disabled={busy}>{busy ? '计算中…' : record ? '重新计算并保存' : '计算并保存'}</button>
      {record && <p className="hint">历史重算沿用该月已保存的目的地快照，不会被今天的计划设置改写。</p>}
    </form>
    {record && <section className="panel stack"><h2>本月建议</h2><div className="metric-grid"><Metric label="收入" value={yuan(record.incomeCents)} /><Metric label="支出" value={yuan(record.expenseTotalCents)} /><Metric label="可投入基数" value={yuan(record.investableBaseCents)} /><Metric label="建议投入" value={yuan(record.recommendedTotalCents)} /></div>
      {record.investableBaseCents === 0 && <div className="notice">收入扣除平台支出与预留金额后无可投入余额，本月建议为 0。</div>}
      <p className="formula">max(收入 − 支出 − 预留, 0) × {percent(record.contributionBps)}，再按 {yuan(record.roundingUnitCents)} 向下取整；目的地尾差按最大余数法稳定分配。</p>
      <div>{record.allocations.map((a, i) => <div className="actual-row" key={a.id}><div><strong>{a.name}</strong><span>建议 {yuan(a.recommendedCents)} · {percent(a.allocationBps)} · 差额 {yuan(a.actualCents - a.recommendedCents)}</span></div><label>实际投入（元）<input type="number" min="0" step="0.01" value={toYuan(a.actualCents)} onChange={e => setRecord({ ...record, allocations: record.allocations.map((x, j) => i === j ? { ...x, actualCents: fromYuan(e.target.value) } : x) })} /></label></div>)}</div>
      <div className="section-head"><span className={`badge ${record.status}`}>状态：{statusLabel(record.status)} · 实际合计 {yuan(record.allocations.reduce((sum, item) => sum + item.actualCents, 0))}</span><button className="button primary" onClick={() => void updateActuals()}>保存实际投入</button></div>
    </section>}
  </div>
}

function History({ plan }: { plan: Plan }) {
  const [items, setItems] = useState<MonthRecord[]>([]); const load = () => get<{ items: MonthRecord[] }>(`/plans/${plan.id}/months`).then(r => setItems(r.items))
  useEffect(() => { void get<{ items: MonthRecord[] }>(`/plans/${plan.id}/months`).then(r => setItems(r.items)) }, [plan.id])
  const remove = async (month: string) => { if (!confirm(`确认删除 ${month} 的完整记录？此操作不可撤销。`)) return; await del(`/plans/${plan.id}/months/${month}`); load() }
  if (!items.length) return <div className="empty panel"><h2>暂无月度记录</h2><p>启用计划后，从“月度记录”录入第一笔收支。</p></div>
  return <section className="panel"><h2>历史记录</h2><div className="history-list">{items.map(item => <article key={item.id}><div><Link to={`../month/${item.month}`}><strong>{item.month}</strong></Link><span>{statusLabel(item.status)}</span></div><div><span>建议 {yuan(item.recommendedTotalCents)}</span><span>实际 {yuan(item.actualTotalCents)}</span><button className="danger-link" onClick={() => void remove(item.month)}>删除</button></div></article>)}</div></section>
}

function Stats({ plan }: { plan: Plan }) {
  const [stats, setStats] = useState<PlanStats | null>(null); useEffect(() => { get<PlanStats>(`/plans/${plan.id}/stats`).then(setStats) }, [plan.id])
  const max = useMemo(() => Math.max(1, ...(stats?.Destinations ?? []).map(d => d.ActualCents)), [stats])
  if (!stats) return <div className="center">正在加载统计…</div>
  return <div className="stack"><div className="metric-grid"><Metric label="累计建议" value={yuan(stats.RecommendedTotalCents)} /><Metric label="累计实际" value={yuan(stats.ActualTotalCents)} /><Metric label="完成率" value={stats.CompletionRate == null ? '不适用' : `${(stats.CompletionRate * 100).toFixed(1)}%`} /></div>
    <section className="panel"><h2>按目的地统计</h2>{(stats.Destinations ?? []).length === 0 ? <p className="muted">暂无实际投入数据。</p> : stats.Destinations?.map(d => <div className="bar-row" key={d.Name}><span>{d.Name}</span><div><i style={{ width: `${Math.max(2, d.ActualCents / max * 100)}%` }} /></div><strong>{yuan(d.ActualCents)}</strong></div>)}</section>
  </div>
}

function Metric({ label, value }: { label: string; value: string }) { return <div className="metric"><span>{label}</span><strong>{value}</strong></div> }
function statusLabel(status: MonthRecord['status']) { return ({ not_required: '无需投入', not_started: '未开始', partial: '部分完成', complete: '已完成' })[status] }
