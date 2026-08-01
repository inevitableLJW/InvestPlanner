import { useEffect, useMemo, useState, type CSSProperties, type FormEvent } from 'react'
import { Link, NavLink, Navigate, Route, Routes, useNavigate, useParams } from 'react-router-dom'
import { AppIcon, TargetIcon } from '../components/AppIcon'
import { APIError, del, get, put } from '../lib/api'
import { fromYuan, percent, toYuan, yuan } from '../lib/format'
import {
  COMMON_INVESTMENT_TARGETS,
  CUSTOM_TARGET_VALUE,
  activeAllocationTotal,
  allocationOptions,
  distributeAllocationsEvenly,
  duplicateActiveTargetNames,
  fillRemainingAllocation,
  investmentTargetSelectValue,
  normalizeInvestmentTargetName,
  reorderDestinations,
  sortDestinationsByAllocation,
} from '../lib/investmentTargets'
import type { Destination, Expense, ExpenseSource, MonthRecord, Plan, PlanStats } from '../types'

function usePlanData() {
  const { planId = '' } = useParams()
  const [plan, setPlan] = useState<Plan | null>(null)
  const [error, setError] = useState('')
  useEffect(() => { void get<Plan>(`/plans/${planId}`).then(result => setPlan({ ...result, destinations: result.destinations.map(item => ({ ...item, name: normalizeInvestmentTargetName(item.name) })) })).catch(reason => setError(reason.message)) }, [planId])
  return { planId, plan, setPlan, error }
}

export function PlanPage() {
  const data = usePlanData()
  if (data.error) return <div className="alert page-feedback" role="alert">{data.error} <Link to="/plans">返回计划列表</Link></div>
  if (!data.plan) return <div className="center loading-state" role="status"><span className="spinner" />正在加载计划…</div>
  const { plan } = data
  const basePath = `/plans/${plan.id}`
  return <>
    <section className="plan-title">
      <div><Link className="back-link" to="/plans">← 所有计划</Link><h1>{plan.name}</h1><p className="muted">把每月结余有节奏地分配到你的长期投资标的。</p></div>
      <span className={`badge ${plan.status}`}><i />{plan.status === 'active' ? '计划运行中' : plan.status === 'archived' ? '已归档' : '待配置'}</span>
    </section>
    <nav className="tabs" aria-label="计划导航">
      <NavLink end to={basePath}>概览</NavLink><NavLink to={`${basePath}/settings`}>计划设置</NavLink><NavLink to={`${basePath}/month`}>月度记录</NavLink><NavLink to={`${basePath}/history`}>历史记录</NavLink><NavLink to={`${basePath}/stats`}>投入统计</NavLink>
    </nav>
    <Routes>
      <Route index element={<Overview plan={plan} />} />
      <Route path="settings" element={<Settings plan={plan} onSaved={data.setPlan} />} />
      <Route path="month" element={<MonthEditor plan={plan} />} />
      <Route path="month/:month" element={<MonthEditor plan={plan} />} />
      <Route path="history" element={<History plan={plan} />} />
      <Route path="stats" element={<Stats plan={plan} />} />
      <Route path="*" element={<Navigate to={basePath} replace />} />
    </Routes>
  </>
}

function Overview({ plan }: { plan: Plan }) {
  const active = plan.destinations.filter(item => item.active && !item.archived)
  const total = activeAllocationTotal(plan.destinations)
  return <section className="stack page-enter">
    {plan.status !== 'active' && <div className="notice notice-action"><div><strong>还差一步就能开始记录</strong><span>至少启用一个投资标的，并让分配比例合计为 100%。</span></div><Link className="button secondary" to="settings">去完成设置</Link></div>}
    <div className="metric-grid">
      <Metric label="已启用标的" value={`${active.length} 个`} note="参与下月计算" />
      <Metric label="分配进度" value={percent(total)} note={total === 10000 ? '配置完整' : `还差 ${percent(Math.abs(10000 - total))}`} />
      <Metric label="每月预留" value={yuan(plan.reserveCents)} note="计算前先扣除" />
      <Metric label="默认投入率" value={percent(plan.defaultContributionBps)} note="每月可单独调整" />
    </div>
    <section className="panel">
      <div className="section-head"><div><p className="section-kicker">ALLOCATION</p><h2>投资标的配置</h2></div><Link className="text-link" to="settings">编辑配置 →</Link></div>
      {active.length ? <div className="overview-allocations">{active.map(item => <div className="allocation-row" key={item.id ?? item.name}><div><TargetIcon name={item.name} /><span>{item.name}</span></div><div><span className="allocation-track"><i style={{ width: `${item.allocationBps / 100}%` }} /></span><strong>{percent(item.allocationBps)}</strong></div></div>)}</div> : <div className="soft-empty"><TargetIcon name="" /><p>尚未启用投资标的</p></div>}
    </section>
  </section>
}

function Settings({ plan, onSaved }: { plan: Plan; onSaved(plan: Plan): void }) {
  const navigate = useNavigate()
  const [draft, setDraft] = useState(plan)
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState(false)
  const [draggingIndex, setDraggingIndex] = useState<number | null>(null)
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null)
  const active = draft.destinations.filter(item => item.active && !item.archived)
  const activeTotal = activeAllocationTotal(draft.destinations)
  const duplicateNames = duplicateActiveTargetNames(draft.destinations)
  const emptyActiveName = active.some(item => !item.name.trim())
  const allocationReady = active.length > 0 && activeTotal === 10000 && duplicateNames.size === 0 && !emptyActiveName
  const updateDestination = (index: number, update: Partial<Destination>) => setDraft(current => ({ ...current, destinations: current.destinations.map((item, itemIndex) => itemIndex === index ? { ...item, ...update } : item) }))
  const addDestination = () => {
    const used = new Set(draft.destinations.map(item => item.name.trim()))
    const nextPreset = COMMON_INVESTMENT_TARGETS.find(name => !used.has(name))
    setDraft(current => ({ ...current, destinations: [...current.destinations, { name: nextPreset ?? '', active: true, archived: false, sortOrder: current.destinations.length, allocationBps: 0 }] }))
  }
  const remove = (index: number) => setDraft(current => ({ ...current, destinations: current.destinations.filter((_, itemIndex) => itemIndex !== index).map((item, itemIndex) => ({ ...item, sortOrder: itemIndex })) }))
  const move = (index: number, delta: number) => setDraft(current => ({ ...current, destinations: reorderDestinations(current.destinations, index, index + delta) }))
  const reorder = (fromIndex: number, toIndex: number) => setDraft(current => ({ ...current, destinations: reorderDestinations(current.destinations, fromIndex, toIndex) }))
  const selectTarget = (index: number, value: string) => {
    const current = draft.destinations[index]
    updateDestination(index, { name: value === CUSTOM_TARGET_VALUE ? (investmentTargetSelectValue(current.name) === CUSTOM_TARGET_VALUE ? current.name : '') : value })
  }
  const save = async (event: FormEvent) => {
    event.preventDefault(); setMessage('')
    if (draft.status === 'active' && !allocationReady) { setMessage('请先修正投资标的配置，再启用计划'); return }
    setBusy(true)
    try {
      const result = await put<Plan>(`/plans/${plan.id}`, {
        name: draft.name, status: draft.status, defaultContributionBps: draft.defaultContributionBps,
        reserveCents: draft.reserveCents, roundingUnitCents: draft.roundingUnitCents, version: draft.version,
        destinations: draft.destinations.map(item => ({ id: item.id ?? '', name: item.name.trim(), active: item.active, allocationBps: item.allocationBps, sortOrder: item.sortOrder, version: item.version ?? 0 })),
      })
      const normalized = { ...result, destinations: result.destinations.map(item => ({ ...item, name: normalizeInvestmentTargetName(item.name) })) }
      setDraft(normalized); onSaved(normalized); setSaved(true)
    } catch (reason) { setMessage(reason instanceof APIError ? reason.message : '保存失败，请稍后重试') }
    finally { setBusy(false) }
  }
  return <form className="stack settings-page page-enter" onSubmit={save}>
    {message && <div className="alert page-feedback" role="alert">{message}</div>}
    <section className="panel settings-section">
      <div className="section-number">01</div>
      <div className="section-intro"><p className="section-kicker">BASICS</p><h2>基本信息</h2><p className="muted">给计划一个清晰的名字，并决定当前是否开始运行。</p></div>
      <div className="form-grid settings-fields">
        <label>计划名称<input required value={draft.name} onChange={event => setDraft({ ...draft, name: event.target.value })} /></label>
        <label>计划状态<select value={draft.status} onChange={event => setDraft({ ...draft, status: event.target.value as Plan['status'] })}><option value="draft">待配置</option><option value="active" disabled={!allocationReady}>运行中</option><option value="archived">已归档</option></select><span className="field-help">{allocationReady ? '标的配置已就绪，可以启用' : '比例达到 100% 后即可启用'}</span></label>
      </div>
    </section>
    <section className="panel settings-section">
      <div className="section-number">02</div>
      <div className="section-intro"><p className="section-kicker">RULES</p><h2>每月投资规则</h2><p className="muted">先扣除支出和预留，再按投入比例生成建议金额。</p></div>
      <div className="form-grid three settings-fields">
        <label>默认投入比例<select value={draft.defaultContributionBps} onChange={event => setDraft({ ...draft, defaultContributionBps: Number(event.target.value) })}>{allocationOptions(draft.defaultContributionBps).map(value => <option value={value * 100} key={value}>{value}%</option>)}</select><span className="field-help">新月份会自动带入</span></label>
        <label>每月预留金额<div className="input-affix"><span>¥</span><input type="number" min="0" step="0.01" value={draft.reserveCents / 100} onChange={event => setDraft({ ...draft, reserveCents: fromYuan(event.target.value) })} /></div><span className="field-help">不会进入本月可投入基数</span></label>
        <label>向下取整单位<div className="input-affix"><span>¥</span><input type="number" min="0.01" step="0.01" value={draft.roundingUnitCents / 100} onChange={event => setDraft({ ...draft, roundingUnitCents: fromYuan(event.target.value) })} /></div><span className="field-help">建议金额按此单位取整</span></label>
      </div>
    </section>
    <section className="panel settings-section targets-section">
      <div className="section-number">03</div>
      <div className="section-intro target-intro"><div><p className="section-kicker">TARGETS</p><h2>投资标的</h2><p className="muted">从常用列表选择即可；只有列表外标的才需要手动输入。</p></div><button type="button" className="button secondary" onClick={addDestination}>＋ 添加投资标的</button></div>
      <div className={`allocation-summary ${allocationReady ? 'valid' : 'invalid'}`}>
        <div className="allocation-summary-copy"><span>分配进度</span><strong>{percent(activeTotal)}</strong><small>{active.length === 0 ? '请至少启用一个标的' : activeTotal === 10000 ? '配置完整，可以保存启用' : activeTotal < 10000 ? `还剩 ${percent(10000 - activeTotal)} 待分配` : `超出 ${percent(activeTotal - 10000)}`}</small></div>
        <div className="allocation-progress" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.min(100, activeTotal / 100)}><i style={{ width: `${Math.min(100, activeTotal / 100)}%` }} /></div>
        <div className="allocation-actions"><button type="button" className="button compact" disabled={!active.length} onClick={() => setDraft(current => ({ ...current, destinations: sortDestinationsByAllocation(current.destinations) }))}>按比例从高到低</button><button type="button" className="button compact" disabled={!active.length} onClick={() => setDraft(current => ({ ...current, destinations: distributeAllocationsEvenly(current.destinations) }))}>智能均分到 100%</button></div>
      </div>
      {duplicateNames.size > 0 && <div className="inline-error" role="alert">存在重复的启用标的，请修改后再保存。</div>}
      <div className="destination-list">{draft.destinations.map((item, index) => {
        const targetValue = investmentTargetSelectValue(item.name)
        const duplicate = item.active && duplicateNames.has(item.name.trim().toLocaleLowerCase())
        const otherTotal = draft.destinations.reduce((total, other, otherIndex) => total + (otherIndex !== index && other.active && !other.archived ? other.allocationBps : 0), 0)
        return <fieldset className={`destination ${item.active ? 'enabled' : ''} ${duplicate ? 'has-error' : ''} ${draggingIndex === index ? 'dragging' : ''} ${dragOverIndex === index && draggingIndex !== index ? 'drag-over' : ''}`} key={item.id ?? `new-${index}`} disabled={item.archived} onDragOver={event => { event.preventDefault(); if (draggingIndex !== null && draggingIndex !== index) setDragOverIndex(index) }} onDrop={event => { event.preventDefault(); const source = Number(event.dataTransfer.getData('text/plain')); if (Number.isInteger(source)) reorder(source, index); setDraggingIndex(null); setDragOverIndex(null) }}>
          <div className="destination-leading"><TargetIcon name={item.name} /><label className="switch"><input type="checkbox" checked={item.active} onChange={event => updateDestination(index, { active: event.target.checked })} /><span /><b>{item.active ? '已启用' : '未启用'}</b></label></div>
          <div className="target-fields">
            <label>标的类型<select aria-label={`投资标的 ${index + 1}`} value={targetValue} onChange={event => selectTarget(index, event.target.value)}>{COMMON_INVESTMENT_TARGETS.map(name => <option key={name} value={name}>{name}</option>)}<option value={CUSTOM_TARGET_VALUE}>＋ 自定义标的</option></select></label>
            {targetValue === CUSTOM_TARGET_VALUE && <label>自定义名称<input aria-label={`自定义标的名称 ${index + 1}`} required value={item.name} placeholder="输入列表外的投资标的" onChange={event => updateDestination(index, { name: event.target.value })} /></label>}
          </div>
          <div className="allocation-field"><label>分配比例<select aria-label={`${item.name || `标的 ${index + 1}`}分配比例`} value={item.allocationBps / 100} onChange={event => updateDestination(index, { allocationBps: Number(event.target.value) * 100 })}>{allocationOptions(item.allocationBps).map(value => <option key={value} value={value}>{value}%</option>)}</select></label><button type="button" className="mini-action" disabled={!item.active || otherTotal > 10000} onClick={() => setDraft({ ...draft, destinations: fillRemainingAllocation(draft.destinations, index) })}>补足剩余</button></div>
          <div className="inline-actions"><button type="button" className="drag-handle" draggable aria-label={`拖动排序 ${item.name || `标的 ${index + 1}`}`} title="拖动调整顺序" onDragStart={event => { event.dataTransfer.effectAllowed = 'move'; event.dataTransfer.setData('text/plain', String(index)); setDraggingIndex(index) }} onDragEnd={() => { setDraggingIndex(null); setDragOverIndex(null) }}>⠿</button><button type="button" aria-label="上移" disabled={index === 0} onClick={() => move(index, -1)}>↑</button><button type="button" aria-label="下移" disabled={index === draft.destinations.length - 1} onClick={() => move(index, 1)}>↓</button><button type="button" className="remove-action" onClick={() => remove(index)}>{item.archived ? '已归档' : '移除'}</button></div>
        </fieldset>
      })}</div>
    </section>
    <div className="save-dock"><div><strong>{allocationReady ? '✓ 标的配置已就绪' : '标的配置尚未完成'}</strong><span>{draft.status === 'draft' ? '可以先保存草稿，完成后再启用。' : allocationReady ? '保存后即可用于月度计算。' : '运行中的计划必须合计为 100%。'}</span></div><button className="button primary" disabled={busy || (draft.status === 'active' && !allocationReady)}>{busy ? <><span className="button-spinner" />保存中…</> : '保存计划设置'}</button></div>
    {saved && <div className="modal-backdrop" role="presentation"><section className="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="saved-settings-title"><span className="dialog-icon">✓</span><h2 id="saved-settings-title">设置已保存</h2><p>投资标的和投资规则已更新。确认后回到计划概览。</p><button type="button" className="button primary wide" onClick={() => { setSaved(false); navigate(`/plans/${plan.id}`, { replace: true }) }}>确认并返回概览</button></section></div>}
  </form>
}

function defaultMonth() { const date = new Date(); return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}` }
function blankExpenses(sources: ExpenseSource[]): Expense[] { return sources.filter(source => source.active).map(source => ({ sourceId: source.id, sourceName: source.name, amountCents: 0, sortOrder: source.sortOrder })) }

function MonthEditor({ plan }: { plan: Plan }) {
  const params = useParams(); const navigate = useNavigate()
  const [month, setMonth] = useState(params.month ?? defaultMonth())
  const [income, setIncome] = useState('')
  const [rate, setRate] = useState(plan.defaultContributionBps / 100)
  const [expenses, setExpenses] = useState<Expense[]>([])
  const [note, setNote] = useState('')
  const [record, setRecord] = useState<MonthRecord | null>(null)
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    setRecord(null); setIncome(''); setRate(plan.defaultContributionBps / 100); setNote(''); setMessage('')
    void get<{ items: ExpenseSource[] }>('/expense-sources').then(result => setExpenses(blankExpenses(result.items)))
    void get<MonthRecord>(`/plans/${plan.id}/months/${month}`).then(result => { setRecord(result); setIncome(toYuan(result.incomeCents)); setRate(result.contributionBps / 100); setExpenses(result.expenses); setNote(result.note) }).catch(reason => { if (reason.status !== 404) setMessage(reason.message); else setRecord(null) })
  }, [plan.id, plan.defaultContributionBps, month])
  const expenseTotal = expenses.reduce((sum, expense) => sum + expense.amountCents, 0)
  const liveIncome = fromYuan(income)
  const estimatedBase = Math.max(liveIncome - expenseTotal - plan.reserveCents, 0)
  const rawRecommended = Math.floor(estimatedBase * rate / 100)
  const estimatedRecommended = Math.floor(rawRecommended / plan.roundingUnitCents) * plan.roundingUnitCents
  const save = async (event: FormEvent) => {
    event.preventDefault()
    if (record && month < defaultMonth() && !window.confirm('这是历史月份。确认使用该月已保存的投资标的快照重新计算？')) return
    setBusy(true); setMessage('')
    try {
      const result = await put<MonthRecord>(`/plans/${plan.id}/months/${month}`, { incomeCents: fromYuan(income), contributionBps: Math.round(rate * 100), expenses, note, ...(record ? { version: record.version } : {}) })
      setRecord(result); setMessage('计算完成，月度记录已保存'); navigate(`/plans/${plan.id}/month/${month}`, { replace: true })
    } catch (reason) { setMessage(reason instanceof APIError ? reason.message : '保存失败，请稍后重试') }
    finally { setBusy(false) }
  }
  const copyLayout = async () => {
    try { const result = await get<{ items: Expense[] }>(`/plans/${plan.id}/months/previous-sources?before=${month}`); setExpenses(result.items.map(item => ({ ...item, amountCents: 0 }))); setMessage('已复制上月支出来源，金额已清空') }
    catch (reason) { setMessage(reason instanceof APIError ? reason.message : '没有可复制的上月布局') }
  }
  const updateActuals = async () => {
    if (!record) return
    try { const result = await put<MonthRecord>(`/plans/${plan.id}/months/${month}/actuals`, { version: record.version, items: record.allocations.map(item => ({ allocationId: item.id, actualCents: item.actualCents })) }); setRecord(result); setMessage('实际投入已更新') }
    catch (reason) { setMessage(reason instanceof APIError ? reason.message : '更新失败，请稍后重试') }
  }
  const investAllByTarget = async () => {
    if (!record) return
    setBusy(true); setMessage('')
    try {
      const result = await put<MonthRecord>(`/plans/${plan.id}/months/${month}/actuals`, { version: record.version, items: record.allocations.map(item => ({ allocationId: item.id, actualCents: item.recommendedCents })) })
      setRecord(result); setMessage('已按每个投资标的的建议金额全部记为实际投入')
    } catch (reason) { setMessage(reason instanceof APIError ? reason.message : '更新失败，请稍后重试') }
    finally { setBusy(false) }
  }
  if (plan.status !== 'active') return <div className="empty panel page-enter"><span className="empty-icon">⌁</span><h2>先完成计划设置</h2><p>启用至少一个投资标的并把比例分配到 100%，就可以开始记录每月收支。</p><Link className="button primary" to="../settings">去设置投资标的</Link></div>
  return <div className="stack page-enter">
    {message && <div className={message.includes('失败') ? 'alert page-feedback' : 'success page-feedback'} role="status">{message}</div>}
    <form className="panel stack month-form" onSubmit={save}>
      <div className="section-head month-heading"><div><p className="section-kicker">MONTHLY CHECK-IN</p><h2>记录本月收支</h2><p className="muted">用上月收入减去本月支出与预留金额，得到本月可投资基数。</p></div><label className="month-picker">记录月份<input type="month" value={month} onChange={event => setMonth(event.target.value)} /></label></div>
      <div className="cashflow-top-grid">
        <label className="income-field">上月总收入<div className="input-affix prominent"><span>¥</span><input type="number" min="0" step="0.01" placeholder="0.00" value={income} onChange={event => setIncome(event.target.value)} /></div><span className="field-help">填写税后到账金额</span></label>
        <div className="rate-field"><span className="field-label">本月投入比例</span><div className="preset-row" aria-label="本月投入比例预设">{[50, 60, 70, 80, 100].map(value => <button type="button" className={rate === value ? 'active' : ''} aria-pressed={rate === value} key={value} onClick={() => setRate(value)}>{value}%</button>)}</div><label className="sr-only" htmlFor="custom-rate">自定义投入比例</label><div className="rate-input"><input id="custom-rate" type="number" min="0" max="100" step="1" value={rate} onChange={event => setRate(Number(event.target.value))} /><span>%</span></div></div>
      </div>
      <div className="section-head expense-heading"><div><h3>支出 App</h3><p className="muted">按付款 App 汇总即可，同一笔消费只记录一次。</p></div><button type="button" className="button secondary compact" onClick={() => void copyLayout()}>↗ 复制上月来源</button></div>
      <div className="expense-grid">{expenses.map((expense, index) => <label className={`expense-card ${expense.amountCents > 0 ? 'has-value' : ''}`} key={expense.sourceId || index}><span className="expense-card-head"><AppIcon name={expense.sourceName} size="lg" /><span><strong>{expense.sourceName}</strong><small>{expense.amountCents > 0 ? '已填写' : '本月支出'}</small></span></span><div className="input-affix"><span>¥</span><input aria-label={`${expense.sourceName}支出金额`} type="number" min="0" step="0.01" placeholder="0.00" value={toYuan(expense.amountCents)} onChange={event => setExpenses(expenses.map((item, itemIndex) => itemIndex === index ? { ...item, amountCents: fromYuan(event.target.value) } : item))} /></div></label>)}</div>
      <div className="live-summary" aria-live="polite"><div><span>收入</span><strong>{yuan(liveIncome)}</strong></div><b>−</b><div><span>支出</span><strong>{yuan(expenseTotal)}</strong></div><b>−</b><div><span>预留</span><strong>{yuan(plan.reserveCents)}</strong></div><b>=</b><div className="highlight"><span>预计可投入</span><strong>{yuan(estimatedRecommended)}</strong></div></div>
      <label>本月备注（选填）<textarea placeholder="例如：奖金月、旅行支出、临时调整投入比例…" value={note} onChange={event => setNote(event.target.value)} /></label>
      <button className="button primary calculate-button" disabled={busy}>{busy ? <><span className="button-spinner" />正在计算…</> : record ? '重新计算并保存' : '生成本月投资建议 →'}</button>
      {record && <p className="hint centered">重新计算历史记录时，会沿用该月保存的投资标的快照，不受当前设置影响。</p>}
    </form>
    {record && <section className="panel stack result-panel">
      <div className="section-head"><div><p className="section-kicker">RECOMMENDATION</p><h2>本月投资建议</h2></div><span className={`badge ${record.status}`}>{statusLabel(record.status)}</span></div>
      <div className="metric-grid"><Metric label="本月收入" value={yuan(record.incomeCents)} /><Metric label="本月支出" value={yuan(record.expenseTotalCents)} /><Metric label="可投入基数" value={yuan(record.investableBaseCents)} /><Metric label="建议投入" value={yuan(record.recommendedTotalCents)} accent /></div>
      {record.investableBaseCents === 0 && <div className="notice">收入扣除支出与预留金额后暂无可投入余额，本月建议为 0。</div>}
      <details className="formula"><summary>查看计算方式</summary><p>max(收入 − 支出 − 预留, 0) × {percent(record.contributionBps)}，再按 {yuan(record.roundingUnitCents)} 向下取整；投资标的尾差按最大余数法稳定分配。</p></details>
      <div className="actual-list">{record.allocations.map((allocation, index) => <div className="actual-row" key={allocation.id}><TargetIcon name={allocation.name} /><div><strong>{allocation.name}</strong><span>建议 {yuan(allocation.recommendedCents)} · 占比 {percent(allocation.allocationBps)}</span></div><label>实际投入<div className="input-affix"><span>¥</span><input type="number" min="0" step="0.01" value={toYuan(allocation.actualCents)} onChange={event => setRecord({ ...record, allocations: record.allocations.map((item, itemIndex) => index === itemIndex ? { ...item, actualCents: fromYuan(event.target.value) } : item) })} /></div><small className={allocation.actualCents - allocation.recommendedCents >= 0 ? 'positive' : ''}>差额 {yuan(allocation.actualCents - allocation.recommendedCents)}</small></label></div>)}</div>
      <div className="result-actions"><div><span>实际投入合计</span><strong>{yuan(record.allocations.reduce((sum, item) => sum + item.actualCents, 0))}</strong></div><div className="actual-actions"><button type="button" className="button secondary" disabled={busy} onClick={() => void investAllByTarget()}>✓ 按建议全部投入</button><button type="button" className="button primary" disabled={busy} onClick={() => void updateActuals()}>保存实际投入</button></div></div>
    </section>}
  </div>
}

function History({ plan }: { plan: Plan }) {
  const [items, setItems] = useState<MonthRecord[]>([])
  const load = () => get<{ items: MonthRecord[] }>(`/plans/${plan.id}/months`).then(result => setItems(result.items))
  useEffect(() => { void get<{ items: MonthRecord[] }>(`/plans/${plan.id}/months`).then(result => setItems(result.items)) }, [plan.id])
  const remove = async (month: string) => { if (!confirm(`确认删除 ${month} 的完整记录？此操作不可撤销。`)) return; await del(`/plans/${plan.id}/months/${month}`); await load() }
  if (!items.length) return <div className="empty panel page-enter"><span className="empty-icon">◷</span><h2>还没有月度记录</h2><p>从当月收入和支出开始，生成第一份投资建议。</p><Link className="button primary" to="../month">记录本月收支</Link></div>
  return <section className="panel page-enter"><div className="section-head"><div><p className="section-kicker">HISTORY</p><h2>历史记录</h2></div><span className="muted">共 {items.length} 个月</span></div><div className="history-list">{items.map(item => <article key={item.id}><div><Link to={`../month/${item.month}`}><strong>{item.month}</strong></Link><span className={`badge ${item.status}`}>{statusLabel(item.status)}</span></div><div><span>建议 <strong>{yuan(item.recommendedTotalCents)}</strong></span><span>实际 <strong>{yuan(item.actualTotalCents)}</strong></span><button className="danger-link" onClick={() => void remove(item.month)}>删除</button></div></article>)}</div></section>
}

function Stats({ plan }: { plan: Plan }) {
  const [stats, setStats] = useState<PlanStats | null>(null)
  useEffect(() => { void get<PlanStats>(`/plans/${plan.id}/stats`).then(setStats) }, [plan.id])
  const max = useMemo(() => Math.max(1, ...(stats?.Destinations ?? []).map(item => item.ActualCents)), [stats])
  const monthly = stats?.Monthly ?? []
  const portfolioTotal = stats?.Destinations?.reduce((sum, item) => sum + item.ActualCents, 0) ?? 0
  if (!stats) return <div className="center loading-state"><span className="spinner" />正在加载统计…</div>
  return <div className="stack page-enter"><div className="metric-grid"><Metric label="累计建议" value={yuan(stats.RecommendedTotalCents)} /><Metric label="累计实际" value={yuan(stats.ActualTotalCents)} accent /><Metric label="投入完成率" value={stats.CompletionRate == null ? '不适用' : `${(stats.CompletionRate * 100).toFixed(1)}%`} /><Metric label="已记录月份" value={`${monthly.length} 个月`} note={monthly.length ? `月均实际 ${yuan(Math.round(stats.ActualTotalCents / monthly.length))}` : '开始记录后显示趋势'} /></div>
    <section className="panel stats-trend"><div className="section-head"><div><p className="section-kicker">MONTHLY TREND</p><h2>每月投入趋势</h2><p className="muted">蓝线为建议投入，实线为实际投入。</p></div><div className="chart-legend"><span><i className="recommended" />建议</span><span><i className="actual" />实际</span></div></div>{monthly.length ? <TrendChart items={monthly} /> : <div className="soft-empty"><p>完成第一条月度记录后，这里会展示投入趋势。</p></div>}</section>
    <div className="stats-bottom-grid"><section className="panel"><div className="section-head"><div><p className="section-kicker">COMPLETION</p><h2>投入完成情况</h2></div></div>{stats.CompletionRate == null ? <div className="soft-empty"><p>暂无完成率数据</p></div> : <div className="completion-chart"><div className="completion-ring" style={{ '--completion': `${Math.min(stats.CompletionRate * 100, 100)}%` } as CSSProperties}><div className="completion-ring-label"><strong>{(stats.CompletionRate * 100).toFixed(1)}%</strong><span>完成率</span></div></div><div className="completion-copy"><strong>{yuan(stats.ActualTotalCents)}</strong><span>实际投入</span><small>目标建议 {yuan(stats.RecommendedTotalCents)}</small></div></div>}</section>
      <section className="panel"><div className="section-head"><div><p className="section-kicker">ALLOCATION</p><h2>资产分布</h2></div></div>{(stats.Destinations ?? []).length === 0 ? <div className="soft-empty"><p>暂无实际投入数据</p></div> : <PortfolioChart items={stats.Destinations ?? []} total={portfolioTotal} />}</section></div>
    <section className="panel"><div className="section-head"><div><p className="section-kicker">PORTFOLIO</p><h2>按投资标的统计</h2></div><span className="muted">按实际投入金额排序</span></div>{(stats.Destinations ?? []).map(item => <div className="bar-row" key={item.Name}><span>{item.Name}</span><div><i style={{ width: `${Math.max(2, item.ActualCents / max * 100)}%` }} /></div><strong>{yuan(item.ActualCents)}</strong></div>)}</section>
  </div>
}

function TrendChart({ items }: { items: NonNullable<PlanStats['Monthly']> }) {
  const width = 720; const height = 280; const padding = { top: 20, right: 22, bottom: 42, left: 76 }
  const rawMax = Math.max(1, ...items.flatMap(item => [item.RecommendedCents, item.ActualCents]))
  const interval = Math.max(10_000, Math.ceil(rawMax / 4 / 10_000) * 10_000)
  const max = interval * 4
  const chartWidth = width - padding.left - padding.right; const chartHeight = height - padding.top - padding.bottom
  const point = (value: number, index: number) => ({ x: padding.left + (items.length === 1 ? chartWidth / 2 : index * chartWidth / (items.length - 1)), y: padding.top + chartHeight - value / max * chartHeight })
  const path = (key: 'RecommendedCents' | 'ActualCents') => items.map((item, index) => { const { x, y } = point(item[key], index); return `${index ? 'L' : 'M'} ${x} ${y}` }).join(' ')
  const ticks = [0, 1, 2, 3, 4]
  return <div className="trend-chart" role="img" aria-label="每月建议投入与实际投入折线图"><svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="xMidYMid meet">{ticks.map(tick => { const value = interval * tick; const y = padding.top + chartHeight - value / max * chartHeight; return <g key={tick}><line className={tick === 0 ? 'chart-axis' : 'chart-grid-line'} x1={padding.left} x2={width - padding.right} y1={y} y2={y} /><text className="chart-y-label" x={padding.left - 11} y={y + 4}>{yuan(value)}</text></g> })}<path className="chart-line recommended" d={path('RecommendedCents')} /><path className="chart-line actual" d={path('ActualCents')} />{items.map((item, index) => { const recommended = point(item.RecommendedCents, index); const actual = point(item.ActualCents, index); return <g key={item.Month}><circle className="chart-dot recommended" cx={recommended.x} cy={recommended.y} r="4"><title>{`${item.Month} · 建议 ${yuan(item.RecommendedCents)}`}</title></circle><circle className="chart-dot actual" cx={actual.x} cy={actual.y} r="4"><title>{`${item.Month} · 实际 ${yuan(item.ActualCents)}`}</title></circle><text className="chart-month" x={actual.x} y={height - 14}>{item.Month.slice(5)}</text></g> })}</svg></div>
}

function PortfolioChart({ items, total }: { items: NonNullable<PlanStats['Destinations']>; total: number }) {
  const colors = ['#286bd6', '#65a4fb', '#8266d7', '#2ab6a5', '#f0ae4e', '#e67374', '#8092b5']
  const safeTotal = Math.max(1, total)
  let cursor = 0
  const segments = items.map((item, index) => { const start = cursor; cursor += item.ActualCents / safeTotal * 100; return `${colors[index % colors.length]} ${start}% ${cursor}%` })
  return <div className="portfolio-chart"><div className="portfolio-donut" style={{ background: `conic-gradient(${segments.join(', ')})` }}><div><strong>{yuan(total)}</strong><span>实际投入</span></div></div><div className="portfolio-legend">{items.slice(0, 5).map((item, index) => <div key={item.Name}><i style={{ background: colors[index % colors.length] }} /><span>{item.Name}</span><strong>{(item.ActualCents / safeTotal * 100).toFixed(1)}%</strong></div>)}</div></div>
}

function Metric({ label, value, note, accent = false }: { label: string; value: string; note?: string; accent?: boolean }) { return <div className={`metric ${accent ? 'accent' : ''}`}><span>{label}</span><strong>{value}</strong>{note && <small>{note}</small>}</div> }
function statusLabel(status: MonthRecord['status']) { return ({ not_required: '无需投入', not_started: '待执行', partial: '部分完成', complete: '已完成' })[status] }
