import { useEffect, useMemo, useState, type CSSProperties, type FormEvent } from 'react'
import { Link, NavLink, Navigate, Route, Routes, useNavigate, useParams } from 'react-router-dom'
import { createPortal } from 'react-dom'
import { AppIcon, TargetIcon } from '../components/AppIcon'
import { APIError, del, get, put } from '../lib/api'
import { fromYuan, parseYuanExpression, percent, toYuan, yuan } from '../lib/format'
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
  presetMarketProfile,
  reorderDestinations,
  sortDestinationsByAllocation,
  withPresetMarketProfile,
} from '../lib/investmentTargets'
import type { Destination, ExecutionMode, Expense, ExpenseSource, FXForecast, MonthRecord, Plan, PlanStats, QDIIResult } from '../types'

function usePlanData() {
  const { planId = '' } = useParams()
  const [plan, setPlan] = useState<Plan | null>(null)
  const [error, setError] = useState('')
  useEffect(() => { void get<Plan>(`/plans/${planId}`).then(result => setPlan({ ...result, destinations: result.destinations.map(item => withPresetMarketProfile({ ...item, name: normalizeInvestmentTargetName(item.name), executionMode: item.executionMode ?? 'unknown', settlementCurrency: item.settlementCurrency ?? 'unknown', benchmarkId: item.benchmarkId ?? '', fxQuotaRelevant: item.fxQuotaRelevant ?? false })) })).catch(reason => setError(reason.message)) }, [planId])
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
    {plan.status === 'draft' && <div className="notice notice-action"><div><strong>计划尚未发布</strong><span>保存只会保留草稿；配置完整并发布后，才可以开始记录每月收支。</span></div><Link className="button secondary" to="settings">去完成并发布</Link></div>}
    {plan.status === 'archived' && <div className="notice"><strong>计划已归档</strong><span>历史记录仍然保留，但不能新增或重新计算月份。</span></div>}
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
  const [busyAction, setBusyAction] = useState<'save_draft' | 'publish' | null>(null)
  const [savedAction, setSavedAction] = useState<'save_draft' | 'publish' | null>(null)
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
    const profile = presetMarketProfile(nextPreset ?? '') ?? { executionMode: 'unknown' as const, settlementCurrency: 'unknown' as const, benchmarkId: '', fxQuotaRelevant: false }
    setDraft(current => ({ ...current, destinations: [...current.destinations, { name: nextPreset ?? '', active: true, archived: false, sortOrder: current.destinations.length, allocationBps: 0, ...profile }] }))
  }
  const remove = (index: number) => setDraft(current => ({ ...current, destinations: current.destinations.filter((_, itemIndex) => itemIndex !== index).map((item, itemIndex) => ({ ...item, sortOrder: itemIndex })) }))
  const move = (index: number, delta: number) => setDraft(current => ({ ...current, destinations: reorderDestinations(current.destinations, index, index + delta) }))
  const reorder = (fromIndex: number, toIndex: number) => setDraft(current => ({ ...current, destinations: reorderDestinations(current.destinations, fromIndex, toIndex) }))
  const selectTarget = (index: number, value: string) => {
    const current = draft.destinations[index]
    if (value === CUSTOM_TARGET_VALUE) {
      updateDestination(index, investmentTargetSelectValue(current.name) === CUSTOM_TARGET_VALUE
        ? { name: current.name }
        : { name: '', executionMode: 'unknown', settlementCurrency: 'unknown', benchmarkId: '', fxQuotaRelevant: false })
      return
    }
    updateDestination(index, { name: value, ...presetMarketProfile(value) })
  }
  const selectExecutionMode = (index: number, mode: ExecutionMode) => {
    const profile = ({ unknown: ['unknown', false], local: ['CNY', false], qdii: ['CNY', false], direct_us: ['USD', true], direct_hk: ['HKD', true], stock_connect: ['CNY', false] } as const)[mode]
    updateDestination(index, { executionMode: mode, settlementCurrency: profile[0], fxQuotaRelevant: profile[1], ...(mode === 'qdii' ? {} : { benchmarkId: '' }) })
  }
  const save = async (event: FormEvent) => {
    event.preventDefault(); setMessage('')
    const submitter = (event.nativeEvent as SubmitEvent).submitter as HTMLButtonElement | null
    const action = submitter?.value === 'publish' ? 'publish' : 'save_draft'
    if (action === 'publish' && !allocationReady) { setMessage('请先修正投资标的配置，再发布计划'); return }
    if (action === 'save_draft' && plan.status === 'active' && !window.confirm('保存为草稿后，计划将暂停运行，不能新增或重新计算月度记录。确认继续？')) return
    setBusyAction(action)
    try {
      const result = await put<Plan>(`/plans/${plan.id}`, {
        name: draft.name, action, defaultContributionBps: draft.defaultContributionBps,
        reserveCents: draft.reserveCents, roundingUnitCents: draft.roundingUnitCents, version: draft.version,
        destinations: draft.destinations.map(item => { const resolved = withPresetMarketProfile(item); return { id: resolved.id ?? '', name: resolved.name.trim(), active: resolved.active, allocationBps: resolved.allocationBps, sortOrder: resolved.sortOrder, version: resolved.version ?? 0, executionMode: resolved.executionMode ?? 'unknown', settlementCurrency: resolved.settlementCurrency ?? 'unknown', benchmarkId: resolved.benchmarkId ?? '', fxQuotaRelevant: resolved.fxQuotaRelevant ?? false } }),
      })
      const normalized = { ...result, destinations: result.destinations.map(item => ({ ...item, name: normalizeInvestmentTargetName(item.name) })) }
      setDraft(normalized); onSaved(normalized); setSavedAction(action)
    } catch (reason) { setMessage(reason instanceof APIError ? reason.message : action === 'publish' ? '发布失败，请稍后重试' : '保存草稿失败，请稍后重试') }
    finally { setBusyAction(null) }
  }
  return <form className="stack settings-page page-enter" onSubmit={save}>
    {message && <div className="alert page-feedback" role="alert">{message}</div>}
    <section className="panel settings-section">
      <div className="section-number">01</div>
      <div className="section-intro"><p className="section-kicker">BASICS</p><h2>基本信息</h2><p className="muted">给计划一个清晰的名字；保存只保留草稿，发布后才开始运行。</p></div>
      <div className="form-grid settings-fields">
        <label>计划名称<input required value={draft.name} onChange={event => setDraft({ ...draft, name: event.target.value })} /></label>
        <div className="status-field"><span>当前状态</span><strong className={`badge ${draft.status}`}><i />{draft.status === 'active' ? '计划运行中' : draft.status === 'archived' ? '已归档' : '草稿，尚未开始'}</strong><span className="field-help">{allocationReady ? '配置已就绪，可以发布' : '比例达到 100% 后即可发布'}</span></div>
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
        <div className="allocation-summary-copy"><span>分配进度</span><strong>{percent(activeTotal)}</strong><small>{active.length === 0 ? '请至少启用一个标的' : activeTotal === 10000 ? '配置完整，可以发布' : activeTotal < 10000 ? `还剩 ${percent(10000 - activeTotal)} 待分配` : `超出 ${percent(activeTotal - 10000)}`}</small></div>
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
            {targetValue === CUSTOM_TARGET_VALUE && <label>执行方式<select aria-label={`${item.name || `标的 ${index + 1}`}执行方式`} value={item.executionMode ?? 'unknown'} onChange={event => selectExecutionMode(index, event.target.value as ExecutionMode)}><option value="unknown">未分类（不推断）</option><option value="local">境内人民币</option><option value="qdii">QDII（人民币申购）</option><option value="direct_us">美股直投（USD）</option><option value="direct_hk">港股直投（HKD）</option><option value="stock_connect">港股通（人民币）</option></select><span className="field-help">结算币种：{item.settlementCurrency ?? 'unknown'} · {item.fxQuotaRelevant ? '计入换汇规划' : '不计入个人换汇'}</span></label>}
            {targetValue === CUSTOM_TARGET_VALUE && item.executionMode === 'qdii' && <label>精确基准 ID<input aria-label={`${item.name || `标的 ${index + 1}`}精确基准 ID`} value={item.benchmarkId ?? ''} placeholder="例如 NASDAQ100_TR" onChange={event => updateDestination(index, { benchmarkId: event.target.value.trim() })} /><span className="field-help">留空时助手显示映射未配置，不按名称猜测。</span></label>}
          </div>
          <div className="allocation-field"><label>分配比例<select aria-label={`${item.name || `标的 ${index + 1}`}分配比例`} value={item.allocationBps / 100} onChange={event => updateDestination(index, { allocationBps: Number(event.target.value) * 100 })}>{allocationOptions(item.allocationBps).map(value => <option key={value} value={value}>{value}%</option>)}</select></label><button type="button" className="mini-action" disabled={!item.active || otherTotal > 10000} onClick={() => setDraft({ ...draft, destinations: fillRemainingAllocation(draft.destinations, index) })}>补足剩余</button></div>
          <div className="inline-actions"><button type="button" className="drag-handle" draggable aria-label={`拖动排序 ${item.name || `标的 ${index + 1}`}`} title="拖动调整顺序" onDragStart={event => { event.dataTransfer.effectAllowed = 'move'; event.dataTransfer.setData('text/plain', String(index)); setDraggingIndex(index) }} onDragEnd={() => { setDraggingIndex(null); setDragOverIndex(null) }}>⠿</button><button type="button" aria-label="上移" disabled={index === 0} onClick={() => move(index, -1)}>↑</button><button type="button" aria-label="下移" disabled={index === draft.destinations.length - 1} onClick={() => move(index, 1)}>↓</button><button type="button" className="remove-action" onClick={() => remove(index)}>{item.archived ? '已归档' : '移除'}</button></div>
        </fieldset>
      })}</div>
    </section>
    <div className="save-dock"><div><strong>{plan.status === 'archived' ? '计划已归档' : allocationReady ? '✓ 标的配置已就绪' : '标的配置尚未完成'}</strong><span>{plan.status === 'archived' ? '归档计划只保留历史，不能修改或重新发布。' : plan.status === 'active' ? '保存草稿会暂停计划；发布会保存并继续运行。' : allocationReady ? '可以保存草稿，只有发布后计划才会开始。' : '未完成也可保存草稿，发布前需分配到 100%。'}</span></div><div className="save-actions"><button className="button secondary" name="action" value="save_draft" disabled={busyAction !== null || plan.status === 'archived'}>{busyAction === 'save_draft' ? <><span className="button-spinner" />保存中…</> : '保存草稿'}</button><button className="button primary" name="action" value="publish" disabled={busyAction !== null || !allocationReady || plan.status === 'archived'}>{busyAction === 'publish' ? <><span className="button-spinner" />发布中…</> : '发布计划'}</button></div></div>
    {savedAction && <div className="modal-backdrop" role="presentation"><section className="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="saved-settings-title"><span className="dialog-icon">✓</span><h2 id="saved-settings-title">{savedAction === 'publish' ? '计划已发布' : '草稿已保存'}</h2><p>{savedAction === 'publish' ? '计划已经开始运行，现在可以记录每月收支。' : '设置已保存为草稿，计划尚未开始运行。'}</p><button type="button" className="button primary wide" onClick={() => { setSavedAction(null); navigate(`/plans/${plan.id}`, { replace: true }) }}>确认并返回概览</button></section></div>}
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
  const [qdiiOpen, setQDIIOpen] = useState(false)
  const [qdiiResult, setQDIIResult] = useState<QDIIResult | null>(null)
  const [qdiiLoading, setQDIILoading] = useState(false)
  const [fxOpen, setFXOpen] = useState(false)
  const [fxPreview, setFXPreview] = useState<FXForecast | null>(null)
  const [fxLoading, setFXLoading] = useState(false)
  const [fxWindow, setFXWindow] = useState<'rolling_12' | 'next_calendar_year'>('rolling_12')
  const [scenarioUSD, setScenarioUSD] = useState('')
  const [scenarioHKD, setScenarioHKD] = useState('')
  useEffect(() => {
    setRecord(null); setIncome(''); setRate(plan.defaultContributionBps / 100); setNote(''); setMessage('')
    void get<{ items: ExpenseSource[] }>('/expense-sources').then(result => setExpenses(blankExpenses(result.items)))
    void get<MonthRecord>(`/plans/${plan.id}/months/${month}`).then(result => { setRecord(result); setIncome(toYuan(result.incomeCents)); setRate(result.contributionBps / 100); setExpenses(result.expenses); setNote(result.note) }).catch(reason => { if (reason.status !== 404) setMessage(reason.message); else setRecord(null) })
  }, [plan.id, plan.defaultContributionBps, month])
  useEffect(() => { void get<FXForecast>(`/plans/${plan.id}/assistants/fx?window=rolling_12`).then(setFXPreview).catch(() => setFXPreview(null)) }, [plan.id])
  const expenseTotal = expenses.reduce((sum, expense) => sum + expense.amountCents, 0)
  const parsedIncome = parseYuanExpression(income)
  const incomeIsValid = parsedIncome !== null
  const liveIncome = parsedIncome ?? 0
  const estimatedBase = Math.max(liveIncome - expenseTotal - plan.reserveCents, 0)
  const save = async (event: FormEvent) => {
    event.preventDefault()
    if (!incomeIsValid) { setMessage('上月总收入仅支持非负金额及 +、- 运算，例如 20000+500-100'); return }
    if (record && month < defaultMonth() && !window.confirm('这是历史月份。确认使用该月已保存的投资标的快照重新计算？')) return
    setBusy(true); setMessage('')
    try {
      const result = await put<MonthRecord>(`/plans/${plan.id}/months/${month}`, { incomeCents: liveIncome, contributionBps: Math.round(rate * 100), expenses, note, ...(record ? { version: record.version } : {}) })
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
  const openQDII = async (allocation: MonthRecord['allocations'][number]) => {
    setQDIIOpen(true); setQDIIResult(null); setQDIILoading(true)
    const remaining = Math.max(allocation.recommendedCents - allocation.actualCents, 0)
    try { setQDIIResult(await get<QDIIResult>(`/plans/${plan.id}/assistants/qdii/${allocation.destinationId}?remainingNeedCents=${remaining}`)) }
    catch (reason) { setQDIIResult({ destinationId: allocation.destinationId, benchmarkId: '', status: 'unavailable', reason: reason instanceof APIError ? reason.message : '助手暂不可用', otc: [], etfs: [] }) }
    finally { setQDIILoading(false) }
  }
  const loadFX = async (window = fxWindow) => {
    setFXLoading(true)
    const query = new URLSearchParams({ window }); if (scenarioUSD) query.set('scenarioUSD', scenarioUSD); if (scenarioHKD) query.set('scenarioHKD', scenarioHKD)
    try { const result = await get<FXForecast>(`/plans/${plan.id}/assistants/fx?${query}`); setFXPreview(result) }
    catch (reason) { setMessage(reason instanceof APIError ? reason.message : '年度换汇预测暂不可用') }
    finally { setFXLoading(false) }
  }
  const openFX = () => { setFXOpen(true); void loadFX() }
  if (plan.status !== 'active') return <div className="empty panel page-enter"><span className="empty-icon">⌁</span><h2>先完成计划设置</h2><p>启用至少一个投资标的并把比例分配到 100%，就可以开始记录每月收支。</p><Link className="button primary" to="../settings">去设置投资标的</Link></div>
  return <div className="stack page-enter">
    {message && <div className={message.includes('失败') ? 'alert page-feedback' : 'success page-feedback'} role="status">{message}</div>}
    <form className="panel stack month-form" onSubmit={save}>
      <div className="section-head month-heading"><div><p className="section-kicker">MONTHLY CHECK-IN</p><h2>记录本月收支</h2><p className="muted">用上月收入减去本月支出与预留金额，得到本月可投资基数。</p></div><label className="month-picker">记录月份<input type="month" value={month} onChange={event => setMonth(event.target.value)} /></label></div>
      <div className="cashflow-top-grid">
        <label className="income-field">上月总收入<div className="input-affix prominent"><span>¥</span><input aria-label="上月总收入" type="text" inputMode="decimal" placeholder="0.00" value={income} onChange={event => setIncome(event.target.value)} aria-invalid={!incomeIsValid} /></div><span className="field-help">{income.trim() && incomeIsValid ? `当前合计 ${yuan(liveIncome)} · 支持 +、- 运算` : '填写税后到账金额；支持 20000+500-100'}</span>{!incomeIsValid && <span className="inline-error">仅支持非负金额及 +、- 运算</span>}</label>
        <div className="rate-field"><span className="field-label">本月投入比例</span><div className="preset-row" aria-label="本月投入比例预设">{[50, 60, 70, 80, 100].map(value => <button type="button" className={rate === value ? 'active' : ''} aria-pressed={rate === value} key={value} onClick={() => setRate(value)}>{value}%</button>)}</div><label className="sr-only" htmlFor="custom-rate">自定义投入比例</label><div className="rate-input"><input id="custom-rate" type="number" min="0" max="100" step="1" value={rate} onChange={event => setRate(Number(event.target.value))} /><span>%</span></div></div>
      </div>
      <div className="section-head expense-heading"><div><h3>支出 App</h3><p className="muted">按付款 App 汇总即可，同一笔消费只记录一次。</p></div><button type="button" className="button secondary compact" onClick={() => void copyLayout()}>↗ 复制上月来源</button></div>
      <div className="expense-grid">{expenses.map((expense, index) => <label className={`expense-card ${expense.amountCents > 0 ? 'has-value' : ''}`} key={expense.sourceId || index}><span className="expense-card-head"><AppIcon name={expense.sourceName} size="lg" /><span><strong>{expense.sourceName}</strong><small>{expense.amountCents > 0 ? '已填写' : '本月支出'}</small></span></span><div className="input-affix"><span>¥</span><input aria-label={`${expense.sourceName}支出金额`} type="number" min="0" step="0.01" placeholder="0.00" value={toYuan(expense.amountCents)} onChange={event => setExpenses(expenses.map((item, itemIndex) => itemIndex === index ? { ...item, amountCents: fromYuan(event.target.value) } : item))} /></div></label>)}</div>
      <div className="live-summary" aria-live="polite"><div><span>收入</span><strong>{yuan(liveIncome)}</strong></div><b>−</b><div><span>支出</span><strong>{yuan(expenseTotal)}</strong></div><b>−</b><div><span>预留</span><strong>{yuan(plan.reserveCents)}</strong></div><b>=</b><div className="highlight"><span>预计可投入</span><strong>{yuan(estimatedBase)}</strong></div></div>
      <label>本月备注（选填）<textarea placeholder="例如：奖金月、旅行支出、临时调整投入比例…" value={note} onChange={event => setNote(event.target.value)} /></label>
      <button className="button primary calculate-button" disabled={busy}>{busy ? <><span className="button-spinner" />正在计算…</> : record ? '重新计算并保存' : '生成本月投资建议 →'}</button>
      {record && <p className="hint centered">重新计算历史记录时，会沿用该月保存的投资标的快照，不受当前设置影响。</p>}
    </form>
    {record && <section className="panel stack result-panel">
      <div className="section-head"><div><p className="section-kicker">RECOMMENDATION</p><h2>本月投资建议</h2></div><span className={`badge ${record.status}`}>{statusLabel(record.status)}</span></div>
      <div className="metric-grid"><Metric label="本月收入" value={yuan(record.incomeCents)} /><Metric label="本月支出" value={yuan(record.expenseTotalCents)} /><Metric label="可投入基数" value={yuan(record.investableBaseCents)} /><Metric label="建议投入" value={yuan(record.recommendedTotalCents)} accent /></div>
      {record.investableBaseCents === 0 && <div className="notice">收入扣除支出与预留金额后暂无可投入余额，本月建议为 0。</div>}
      <details className="formula"><summary>查看计算方式</summary><p>max(收入 − 支出 − 预留, 0) × {percent(record.contributionBps)}；{record.allocations.some(item => item.name.trim() === '现金') ? `非现金投资标的按 ${yuan(record.roundingUnitCents)} 向下取整，剩余尾差（包括零头）全部归入现金。` : `未启用现金时，建议总额按 ${yuan(record.roundingUnitCents)} 向下取整，投资标的尾差按最大余数法稳定分配。`}</p></details>
      <AllocationList plan={plan} record={record} onRecord={setRecord} onQDII={openQDII} onFX={openFX} />
      <div className="result-actions"><div><span>实际投入合计</span><strong>{yuan(record.allocations.reduce((sum, item) => sum + item.actualCents, 0))}</strong></div><div className="actual-actions"><button type="button" className="button secondary" disabled={busy} onClick={() => void investAllByTarget()}>✓ 按建议全部投入</button><button type="button" className="button primary" disabled={busy} onClick={() => void updateActuals()}>保存实际投入</button></div></div>
    </section>}
    {qdiiOpen && createPortal(<QDIIDrawer loading={qdiiLoading} result={qdiiResult} onClose={() => setQDIIOpen(false)} />, document.body)}
    {fxOpen && createPortal(<FXDrawer loading={fxLoading} result={fxPreview} window={fxWindow} scenarioUSD={scenarioUSD} scenarioHKD={scenarioHKD} onWindow={value => { setFXWindow(value); void loadFX(value) }} onScenarioUSD={setScenarioUSD} onScenarioHKD={setScenarioHKD} onApply={() => void loadFX()} onClose={() => setFXOpen(false)} />, document.body)}
  </div>
}

function AllocationList({ plan, record, onRecord, onQDII, onFX }: { plan: Plan; record: MonthRecord; onRecord(record: MonthRecord): void; onQDII(allocation: MonthRecord['allocations'][number]): void; onFX(): void }) {
  const profiles = new Map(plan.destinations.map(item => [item.id, item]))
  const overseas = record.allocations.filter(item => { const profile = profiles.get(item.destinationId); return profile?.fxQuotaRelevant && (profile.executionMode === 'direct_us' || profile.executionMode === 'direct_hk') })
  const usCount = overseas.filter(item => profiles.get(item.destinationId)?.executionMode === 'direct_us').length
  const hkCount = overseas.filter(item => profiles.get(item.destinationId)?.executionMode === 'direct_hk').length
  const marketCounts = [`${usCount ? `美股 ${usCount} 个标的` : ''}`, `${hkCount ? `港股 ${hkCount} 个标的` : ''}`].filter(Boolean).join(' · ')
  const overseasIDs = new Set(overseas.map(item => item.id)); let grouped = false
  const row = (allocation: MonthRecord['allocations'][number]) => {
    const index = record.allocations.findIndex(item => item.id === allocation.id); const profile = profiles.get(allocation.destinationId)
    return <div className="actual-row" key={allocation.id}><TargetIcon name={allocation.name} /><div><strong>{allocation.name}</strong><span>建议 {yuan(allocation.recommendedCents)} · 占比 {percent(allocation.allocationBps)}</span>{profile?.executionMode === 'qdii' && <button type="button" className="assistant-link" onClick={() => onQDII(allocation)}>查看同指数 QDII 基金助手</button>}</div><label>实际投入<div className="input-affix"><span>¥</span><input type="number" min="0" step="0.01" value={toYuan(allocation.actualCents)} onChange={event => onRecord({ ...record, allocations: record.allocations.map((item, itemIndex) => index === itemIndex ? { ...item, actualCents: fromYuan(event.target.value) } : item) })} /></div><small className={allocation.actualCents - allocation.recommendedCents >= 0 ? 'positive' : ''}>差额 {yuan(allocation.actualCents - allocation.recommendedCents)}</small></label></div>
  }
  return <div className="actual-list">{record.allocations.map(allocation => {
    if (!overseasIDs.has(allocation.id)) return row(allocation)
    if (grouped) return null; grouped = true
    return <section className="overseas-group" key="overseas-direct"><div className="overseas-group-title"><div><span>OVERSEAS DIRECT</span><strong>海外直投标的</strong></div><small>{marketCounts}</small></div>{overseas.map(row)}<div className="overseas-group-action"><span>汇总 {overseas.length} 个海外标的</span><button type="button" className="button secondary compact" onClick={onFX}>查看换汇预测</button></div></section>
  })}</div>
}

function DataValue({ value, money = false }: { value: { value: unknown; display: string; reason?: string; freshness: string }; money?: boolean }) {
  const display = value.value == null ? '--' : money && typeof value.value === 'number' ? yuan(value.value) : value.display
  return <span className={`data-value ${value.freshness}`} title={value.reason}>{display}{value.freshness === 'stale' && <small>陈旧</small>}</span>
}
function TrackingErrorValue({ value }: { value: QDIIResult['otc'][number]['trackingError'] }) {
  const display = typeof value.value === 'number' ? `${(value.value * 100).toFixed(2)}%` : '--'
  return <span className={`data-value ${value.freshness}`} title={value.reason}>{display}{value.freshness === 'stale' && <small>陈旧</small>}</span>
}
function trackingErrorFootnote(result: QDIIResult) {
  const value = [...result.otc, ...result.etfs].map(item => item.trackingError).find(item => item.value != null)
  if (!value) return ''
  const details = value.display.match(/相对\s+(QQQ|SPY).*?(\d+)\s*个共同观测日.*?(\d{4}-\d{2}-\d{2})/)
  if (details) return `* 年化跟踪误差按最近一年共同观测日计算；本表相对 ${details[1]}，共 ${details[2]} 个共同观测日，数据截至 ${details[3]}。`
  const observedAt = value.observedAt?.slice(0, 10)
  return `* 年化跟踪误差按最近一年共同观测日计算${observedAt ? `，数据截至 ${observedAt}` : ''}。`
}
function fundShareClassSuffix(value: string) {
  const shareClass = value.trim()
  return shareClass && shareClass.toLowerCase() !== 'unknown' ? ` · ${shareClass} 类` : ''
}
function QDIIDrawer({ loading, result, onClose }: { loading: boolean; result: QDIIResult | null; onClose(): void }) {
  const [selectedOTC, setSelectedOTC] = useState<Set<string>>(new Set())
  const [selectedETF, setSelectedETF] = useState('')
  const [expandedOTC, setExpandedOTC] = useState(false)
  useEffect(() => {
    const candidates = result?.otc ?? []
    const defaults = candidates.filter(item => item.purchaseStatus === 'open').slice(0, 2)
    setSelectedOTC(new Set((defaults.length ? defaults : candidates.slice(0, 2)).map(item => item.fundCode)))
    setSelectedETF(result?.etfs?.[0]?.fundCode ?? '')
    setExpandedOTC(false)
  }, [result])
  const otc = result?.otc ?? []
  const etfs = result?.etfs ?? []
  const shownOTC = expandedOTC ? otc : otc.slice(0, 4)
  const selectedCandidates = otc.filter(item => selectedOTC.has(item.fundCode))
  const capacityKnown = selectedCandidates.every(item => item.executableCapacityCents.value != null)
  const selectedCapacity = capacityKnown ? selectedCandidates.reduce((sum, item) => sum + (item.executableCapacityCents.value ?? 0), 0) : null
  const target = otc[0]?.remainingNeedCents ?? 0
  const gap = selectedCapacity == null ? null : Math.max(0, target - selectedCapacity)
  const selectableOTC = otc.filter(item => item.purchaseStatus !== 'suspended')
  const allSelected = selectableOTC.length > 0 && selectableOTC.every(item => selectedOTC.has(item.fundCode))
  const trackingReason = otc.some(item => item.trackingError.value != null) ? undefined : otc.find(item => item.trackingError.value == null)?.trackingError.reason
  const trackingFootnote = result ? trackingErrorFootnote(result) : ''
  const etfUsesMarketPrice = etfs.some(item => item.trackingError.value != null && item.trackingError.display.includes('场内价格口径'))
  const toggleOTC = (code: string) => { if (otc.find(item => item.fundCode === code)?.purchaseStatus === 'suspended') return; setSelectedOTC(current => { const next = new Set(current); if (next.has(code)) next.delete(code); else next.add(code); return next }) }
  const toggleAll = () => setSelectedOTC(allSelected ? new Set() : new Set(selectableOTC.map(item => item.fundCode)))
  return <div className="drawer-backdrop" role="presentation"><aside className="assistant-drawer qdii-drawer" role="dialog" aria-modal="true" aria-labelledby="qdii-title"><header><div><p className="section-kicker">QDII ASSISTANT</p><h2 id="qdii-title">同指数基金助手</h2><span>场外多选承接，场内 ETF 单选补足缺口</span></div><button type="button" aria-label="关闭 QDII 基金助手" onClick={onClose}>×</button></header>{loading ? <div className="drawer-loading"><span className="spinner" />正在读取最近快照…</div> : result?.status === 'unavailable' ? <div className="assistant-unavailable"><strong>--</strong><p>{result.reason || '市场数据暂不可用'}</p><span>月度实际投入仍可正常填写和保存。</span></div> : <div className="drawer-body qdii-body">
    {trackingReason && <div className="data-notice"><strong>年化跟踪误差暂显示 --</strong><span>{trackingReason}</span></div>}
    <section className="qdii-step"><div className="qdii-step-title"><b>1</b><div><h3>场外基金（多选）</h3><span>按真实渠道限额估算本月承接容量</span></div><strong>本月目标 {yuan(target)}</strong></div>
      <div className="qdii-controls"><label><input type="checkbox" checked={allSelected} onChange={toggleAll} /> 全选</label><span>已选 {selectedOTC.size} 只</span><small>排序：申购状态 · 跟踪误差 · 基金代码</small></div>
      <div className="qdii-table-wrap"><table className="qdii-table"><thead><tr><th>选择</th><th>基金</th><th>当前单日限额</th><th>年化跟踪误差</th><th>剩余可申购日</th><th>本月预计可投</th></tr></thead><tbody>{shownOTC.map(item => <tr key={item.fundCode} className={`${selectedOTC.has(item.fundCode) ? 'selected' : ''} ${item.purchaseStatus === 'suspended' ? 'disabled' : ''}`}><td><input type="checkbox" aria-label={`选择 ${item.name}`} checked={selectedOTC.has(item.fundCode)} disabled={item.purchaseStatus === 'suspended'} onChange={() => toggleOTC(item.fundCode)} /></td><td><strong>{item.name}</strong><span>{item.fundCode}{fundShareClassSuffix(item.shareClass)}</span><b className={`purchase ${item.purchaseStatus}`}>{item.purchaseStatus === 'open' ? '可申购' : item.purchaseStatus === 'suspended' ? '暂停申购' : '状态 --'}</b></td><td><DataValue value={item.dailyLimitCents} money /></td><td><TrackingErrorValue value={item.trackingError} /></td><td><DataValue value={item.remainingPurchaseDays} /></td><td><DataValue value={item.executableCapacityCents} money /></td></tr>)}</tbody></table></div>
      {otc.length > 4 && <button type="button" className="qdii-expand" onClick={() => setExpandedOTC(value => !value)}>{expandedOTC ? '收起候选' : `查看全部（${otc.length}）`}</button>}
      <div className="qdii-table-notes">{trackingFootnote && <small>{trackingFootnote}</small>}<small>* 当前单日限额为公开销售渠道观察值，仅供参考，以实际销售平台为准。</small></div>
      <div className="qdii-total"><span>已选场外合计</span><strong>{selectedCapacity == null ? '--' : yuan(selectedCapacity)}</strong></div>
      <div className="qdii-gap"><span>场内需承接</span><strong>{gap == null ? '--' : yuan(gap)}</strong></div>
    </section>
    <section className="qdii-step"><div className="qdii-step-title"><b>2</b><div><h3>场内承接</h3><span>单选一只精确映射 ETF 承接场外剩余缺口</span></div></div>
      {etfs.length ? <div className="qdii-table-wrap"><table className="qdii-table etf-table"><thead><tr><th>选择</th><th>ETF 名称</th><th>价格</th><th>IOPV</th><th>溢价率</th><th>近一年年化跟踪误差</th></tr></thead><tbody>{etfs.map((item, index) => <tr key={item.fundCode} className={selectedETF === item.fundCode ? 'selected' : ''}><td><input type="radio" name="qdii-etf" aria-label={`选择 ${item.name}`} checked={selectedETF === item.fundCode} onChange={() => setSelectedETF(item.fundCode)} /></td><td><strong>{item.name}</strong><span>{item.fundCode}</span>{index === 0 && item.premiumRate.value != null && <b className="recommendation">低溢价率</b>}</td><td><DataValue value={item.price} /></td><td><DataValue value={item.iopv} /></td><td><DataValue value={item.premiumRate} /></td><td><TrackingErrorValue value={item.trackingError} /></td></tr>)}</tbody></table></div> : <div className="soft-empty"><p>-- · 暂无可用的精确映射 ETF 快照</p></div>}
      {etfUsesMarketPrice && <div className="qdii-table-notes"><small>* 场内结果暂按交易价格相对 QQQ/SPY 计算，包含溢价波动；不等同于基金净值口径。</small></div>}
      <div className="qdii-equation"><span>场外 <strong>{selectedCapacity == null ? '--' : yuan(selectedCapacity)}</strong></span><b>+</b><span>场内 <strong>{gap == null ? '--' : yuan(gap)}</strong></span><b>=</b><span>本月目标 <strong>{yuan(target)}</strong></span></div>
    </section>
  </div>}</aside></div>
}

function FXChart({ result }: { result: FXForecast }) { const points = result.months.filter(item => item.currency === result.currencies[0]); const known = points.map(item => item.cumulativeUsdEquivalent); const max = Math.max(1, ...known.map(value => value ?? 0)); const path = known.map((value, index) => value == null ? '' : `${index ? 'L' : 'M'} ${20 + index * 480 / Math.max(1, known.length - 1)} ${180 - value / max * 150}`).filter(Boolean).join(' '); return <div className="fx-chart" role="img" aria-label="年度累计 USD 等值需求曲线"><svg viewBox="0 0 520 210"><line x1="20" x2="500" y1="180" y2="180" /><path d={path} />{known.map((value,index) => value == null ? null : <circle key={index} cx={20 + index * 480 / Math.max(1, known.length - 1)} cy={180 - value / max * 150} r="3" />)}</svg></div> }
function FXDrawer({ loading, result, window, scenarioUSD, scenarioHKD, onWindow, onScenarioUSD, onScenarioHKD, onApply, onClose }: { loading: boolean; result: FXForecast | null; window: 'rolling_12' | 'next_calendar_year'; scenarioUSD: string; scenarioHKD: string; onWindow(value: 'rolling_12' | 'next_calendar_year'): void; onScenarioUSD(value: string): void; onScenarioHKD(value: string): void; onApply(): void; onClose(): void }) { return <div className="drawer-backdrop" role="presentation"><aside className="assistant-drawer fx-drawer" role="dialog" aria-modal="true" aria-labelledby="fx-title"><header><div><p className="section-kicker">FX FORECAST</p><h2 id="fx-title">年度换汇预测</h2><span>从服务端权威月度建议派生</span></div><button type="button" aria-label="关闭年度换汇预测" onClick={onClose}>×</button></header><div className="drawer-body"><div className="window-switch" aria-label="预测窗口"><button className={window === 'rolling_12' ? 'active' : ''} onClick={() => onWindow('rolling_12')}>滚动 12 个月</button><button className={window === 'next_calendar_year' ? 'active' : ''} onClick={() => onWindow('next_calendar_year')}>下一自然年</button></div>{loading ? <div className="drawer-loading"><span className="spinner" />正在生成预测…</div> : result?.status === 'unavailable' ? <div className="assistant-unavailable"><strong>--</strong><p>{result.reason || '年度换汇预测暂不可用'}</p></div> : !result?.currencies.length ? <div className="assistant-unavailable"><strong>--</strong><p>计划没有需个人换汇的海外直投标的</p></div> : <><div className="fx-summaries">{result.summaries.map(item => <article key={item.currency}><span>{item.currency} 需求</span><strong>{item.foreignAmount == null ? '--' : item.foreignAmount.toLocaleString('zh-CN', { maximumFractionDigits: 2 })}</strong><small>{item.cnyCents == null ? '--' : yuan(item.cnyCents)}</small></article>)}</div><FXChart result={result} /><div className="scenario-grid">{result.currencies.includes('USD') && <label>USD 情景汇率<input type="number" min="0.0001" step="0.0001" placeholder="使用最新中间价" value={scenarioUSD} onChange={event => onScenarioUSD(event.target.value)} /></label>}{result.currencies.includes('HKD') && <label>HKD 情景汇率<input type="number" min="0.0001" step="0.0001" placeholder="使用最新中间价" value={scenarioHKD} onChange={event => onScenarioHKD(event.target.value)} /></label>}<button type="button" className="button secondary" onClick={onApply}>应用情景</button></div><div className="fx-months">{result.months.map(item => <div key={`${item.month}-${item.currency}`}><span>{item.month} · {item.currency}</span><strong>{item.foreignAmount == null ? '--' : item.foreignAmount.toLocaleString('zh-CN', { maximumFractionDigits: 2 })}</strong><small>{item.cnyCents == null ? item.reason || '--' : yuan(item.cnyCents)}</small></div>)}</div><section className="quota-notice"><strong>年度便利化额度参考：{result.quotaUsd.toLocaleString()} USD</strong><span>来源日期：{result.quotaEffectiveDate || '--'}</span><p>{result.complianceNotice}</p>{result.quotaSource && <a href={result.quotaSource} target="_blank" rel="noreferrer">查看官方来源</a>}</section></>}</div></aside></div> }

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
