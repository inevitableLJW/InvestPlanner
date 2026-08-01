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
    <section className="hero-row plans-hero">
      <div><p className="eyebrow">LONG-TERM INVESTING</p><h1>让每月结余，<br /><span>有计划地增长。</span></h1><p className="muted">建立你的投资规则，按 App 汇总支出，把可投入资金分配给长期标的。</p></div>
      <div className="hero-visual" aria-hidden="true"><span>本月计划投入</span><strong>¥ 8,600</strong><div><i style={{ width: '68%' }} /></div><small>按照计划，稳步前进</small></div>
    </section>
    {error && <div className="alert" role="alert">{error}</div>}
    <form className="panel create-row create-plan-card" onSubmit={create}>
      <div className="create-plan-intro"><span>＋</span><div><strong>创建新的定投计划</strong><small>例如：长期资产配置、退休账户、旅行基金</small></div></div>
      <label className="grow"><span className="sr-only">计划名称</span><input id="create-plan" placeholder="给计划起个名字" required value={name} onChange={e => setName(e.target.value)} /></label>
      <button className="button primary" disabled={creating}>{creating ? '创建中…' : '创建计划 →'}</button>
    </form>
    <div className="list-heading"><div><p className="section-kicker">YOUR PLANS</p><h2>我的计划</h2></div><span>{plans.length} 个计划</span></div>
    {plans.length === 0 ? <section className="empty panel"><span className="empty-icon">↗</span><h2>创建你的第一份计划</h2><p>可直接选择现金、债券类基金、纳斯达克100指数、标普500指数、沪深300和中证500等常用投资标的，也支持添加自定义标的。</p><button className="button primary" onClick={() => document.getElementById('create-plan')?.focus()}>创建定投计划</button></section> :
      <section className="card-grid" aria-label="计划列表">{plans.map(plan => <Link className="plan-card" to={`/plans/${plan.id}`} key={plan.id}>
        <div className="card-head"><span className={`badge ${plan.status}`}><i />{plan.status === 'active' ? '运行中' : plan.status === 'archived' ? '已归档' : '待配置'}</span><span className="card-arrow">↗</span></div>
        <h2>{plan.name}</h2>
        <p>{plan.destinations.filter(d => d.active && !d.archived).length} 个投资标的 · 默认投入 {percent(plan.defaultContributionBps)}</p>
        <dl><div><dt>累计建议</dt><dd>{yuan(plan.summary?.RecommendedTotalCents ?? 0)}</dd></div><div><dt>累计实际</dt><dd>{yuan(plan.summary?.ActualTotalCents ?? 0)}</dd></div></dl><div className="card-progress"><i style={{ width: `${Math.min(100, ((plan.summary?.ActualTotalCents ?? 0) / Math.max(1, plan.summary?.RecommendedTotalCents ?? 0)) * 100)}%` }} /></div>
      </Link>)}</section>}
  </>
}
