## ADDED Requirements

### Requirement: Record actual contributions by destination
系统 SHALL 允许用户为每个已计算计划月份的全部目的地快照分别登记非负实际投入金额，并允许实际金额与建议金额不同；现金与其他目的地使用相同行为。

#### Scenario: Save actual contributions
- **WHEN** 用户为该计划月份的一个或多个目的地输入有效实际金额并保存
- **THEN** 系统保存每项目的地实际金额，并展示每项及合计相对建议金额的差额

#### Scenario: Record cash allocation
- **WHEN** 月度快照包含现金目的地且用户登记实际保留现金金额
- **THEN** 系统把该金额计入现金目的地实际金额和该计划统计，不要求证券交易信息

#### Scenario: Record a contribution when the recommendation is zero
- **WHEN** 该月建议定投总额为零但用户仍输入实际金额
- **THEN** 系统允许保存记录并明确显示该金额超出当月建议

#### Scenario: Reject an invalid actual amount
- **WHEN** 用户输入负数、无法解析的金额或超过两位小数的人民币金额
- **THEN** 系统 SHALL 阻止保存并标记无效字段

### Requirement: Derive a monthly execution status
系统 SHALL 根据该计划月份的建议总额和所有目的地实际金额合计派生执行状态，而不是要求用户手动维护状态。

#### Scenario: No contribution is recommended
- **WHEN** 建议总额为零且实际合计也为零
- **THEN** 系统将状态显示为“无需定投”

#### Scenario: Recommended contribution has not started
- **WHEN** 建议总额大于零且实际合计为零
- **THEN** 系统将状态显示为“未执行”

#### Scenario: Contribution is partially completed
- **WHEN** 实际合计大于零但小于建议总额
- **THEN** 系统将状态显示为“部分完成”

#### Scenario: Contribution meets or exceeds the recommendation
- **WHEN** 实际合计大于或等于建议总额且不同时为零
- **THEN** 系统将状态显示为“已完成”

### Requirement: Browse and correct history within a plan
系统 SHALL 在计划详情按月份倒序展示该计划的收入、支出、建议投入、实际投入和执行状态，并允许用户打开、编辑或删除一条月度记录。

#### Scenario: Open a historical plan month
- **WHEN** 用户从计划历史列表选择一个月份
- **THEN** 系统展示该计划该月现金流、计划与目的地快照、分项建议、实际金额和备注

#### Scenario: Delete a monthly record
- **WHEN** 用户请求删除并在确认提示中再次确认
- **THEN** 系统删除该计划该月输入、快照和实际金额，并从该计划统计中移除

### Requirement: Present plan-scoped contribution statistics
系统 SHALL 基于选定计划的月度记录展示每月建议与实际金额、累计实际金额、所有历史目的地的累计实际金额与占比，以及金额完成率。

#### Scenario: View a plan dashboard
- **WHEN** 用户打开一个至少存在一条月度记录的计划仪表盘
- **THEN** 系统只聚合该计划数据，并按目的地快照名称展示累计金额和占比

#### Scenario: Include custom and cash destinations
- **WHEN** 计划历史包含自定义目的地或现金目的地
- **THEN** 系统 SHALL 将它们作为独立目的地纳入趋势、累计金额和占比

#### Scenario: Calculate the monetary completion rate
- **WHEN** 计划统计范围内累计建议投入大于零
- **THEN** 系统将金额完成率计算为该计划累计实际金额除以累计建议金额，并允许结果高于 100%

#### Scenario: No recommended amount in the plan range
- **WHEN** 计划统计范围内累计建议投入为零
- **THEN** 系统将金额完成率显示为“不适用”，而不是显示零或产生除零错误

### Requirement: Keep historical statistics stable after plan changes
系统 SHALL 从每月保存的目的地快照和实际金额生成计划历史统计，当前目的地变化不得追溯改写旧月份。

#### Scenario: Update current destinations
- **WHEN** 用户修改当前计划的目的地名称、启用状态、顺序或比例后返回统计页面
- **THEN** 系统保持所有旧月份的建议和实际数据不变，并使用快照名称展示历史目的地

### Requirement: Present an account plan overview without merging cashflows
系统 SHALL 在账户首页展示每个计划的摘要，但不得把多个计划的收入和支出相加为账号总收支。

#### Scenario: View multiple plans
- **WHEN** 用户拥有多个包含同月份记录的计划
- **THEN** 系统分别展示各计划摘要，不生成可能重复计算收入或支出的账号合计

### Requirement: Scope history and statistics to the authenticated account and plan
系统 SHALL 只使用当前登录用户拥有的选定计划数据生成历史、详情和统计。

#### Scenario: Aggregate selected plan data
- **WHEN** 已登录用户打开一个计划仪表盘
- **THEN** 系统的合计、趋势、占比和完成率只包含该计划的数据

#### Scenario: Request another plan history record
- **WHEN** 用户使用另一个账号或另一个计划的月度记录标识请求详情或删除
- **THEN** 系统 SHALL 按记录不存在处理，且目标计划历史与统计保持不变
