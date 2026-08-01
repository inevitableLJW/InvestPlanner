import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { APIError, get, post } from '../lib/api'
import { percent, yuan } from '../lib/format'
import type { Plan } from '../types'

export function PlansPage() {
  const navigate = useNavigate()
  const [plans, setPlans] = useState<Plan[]>([])
  const [name, setName] = useState('')
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const load = () => get<{ items: Plan[] }>('/plans').then(r => setPlans(r.items)).catch(e => setError(e.message))
  useEffect(() => { void load() }, [])
  const create = async (event: FormEvent) => {
    event.preventDefault(); setCreating(true); setError('')
    try { const plan = await post<Plan>('/plans', { name }); setPlans([plan, ...plans]); setName(''); navigate(`/plans/${plan.id}/settings`) }
    catch (reason) { setError(reason instanceof APIError ? reason.message : '创建失败') }
    finally { setCreating(false) }
  }
  return <>
    <section className="hero-row">
      <div><p className="eyebrow">计划优先</p><h1>我的定投计划</h1><p className="muted">先创建并设置计划，再在计划内填写每月收入和按付款平台汇总的支出。</p></div>
      <button className="button primary" onClick={() => document.getElementById('create-plan')?.focus()}>创建定投计划</button>
    </section>
    {error && <div className="alert" role="alert">{error}</div>}
    <form className="panel create-row" onSubmit={create}>
      <label className="grow">计划名称<input id="create-plan" placeholder="例如：长期资产配置" required value={name} onChange={e => setName(e.target.value)} /></label>
      <button className="button primary" disabled={creating}>{creating ? '创建中…' : '创建计划'}</button>
    </form>
    {plans.length === 0 ? <section className="empty panel"><h2>还没有计划</h2><p>创建后会得到支付宝基金、A股、港美股、现金四个可编辑目的地，它们只是模板，均非必选。</p></section> :
      <section className="card-grid" aria-label="计划列表">{plans.map(plan => <Link className="plan-card" to={`/plans/${plan.id}`} key={plan.id}>
        <div className="card-head"><span className={`badge ${plan.status}`}>{plan.status === 'active' ? '已启用' : '草稿'}</span><span>版本 {plan.version}</span></div>
        <h2>{plan.name}</h2>
        <p>{plan.destinations.filter(d => d.active && !d.archived).length} 个启用目的地 · 默认投入 {percent(plan.defaultContributionBps)}</p>
        <dl><div><dt>累计建议</dt><dd>{yuan(plan.summary?.RecommendedTotalCents ?? 0)}</dd></div><div><dt>累计实际</dt><dd>{yuan(plan.summary?.ActualTotalCents ?? 0)}</dd></div></dl>
      </Link>)}</section>}
  </>
}
