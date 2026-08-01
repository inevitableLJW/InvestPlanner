## MODIFIED Requirements

### Requirement: Create and list investment plans
系统 SHALL 允许每个已登录用户创建多个定投计划，并把计划作为投资标的、月度收支、计算、实际投入、历史和统计的一级容器。

#### Scenario: Create a plan
- **WHEN** 用户输入非空计划名称并创建计划
- **THEN** 系统保存一个归属于该用户的草稿计划，加入可编辑的初始投资标的，并导航到计划详情设置页

#### Scenario: List owned plans
- **WHEN** 已登录用户打开计划首页
- **THEN** 系统只列出该用户的计划及每个计划的状态和摘要

#### Scenario: Use the same month in different plans
- **WHEN** 同一用户的两个计划分别保存相同的 `YYYY-MM` 月份
- **THEN** 系统分别保存两条互不覆盖的计划月度记录

### Requirement: Manage configurable contribution destinations
系统 SHALL 在界面中将 contribution destination 统一称为“投资标的”，允许用户从现金、债券类基金、纳斯达克100指数（QDII）、标普500指数（QDII）、红利低波、A股主动基金、沪深300指数基金、中证500指数基金、A股、港股、美股、纳斯达克100指数（美股）、标普500指数（美股）常用列表选择，或通过“自定义标的”新增列表外名称，并允许排序、启用和停用标的。

#### Scenario: Add a common investment target
- **WHEN** 用户新增标的并选择常用列表中的一项
- **THEN** 系统 SHALL 直接使用该预设名称，不要求用户重复手动输入

#### Scenario: Add a custom investment target
- **WHEN** 用户选择“自定义标的”并输入一个非空且不与该计划其他启用标的重名的名称
- **THEN** 系统保存该自定义标的并允许用户为其设置分配比例

#### Scenario: Hide manual name entry for common targets
- **WHEN** 用户选择常用投资标的
- **THEN** 系统 SHALL 隐藏自定义名称输入框并以所选预设作为标的名称

#### Scenario: Disable an investment target
- **WHEN** 用户停用任意投资标的
- **THEN** 系统不再把该标的用于新月份计算，且 SHALL 保留其他标的配置

#### Scenario: Use only one investment target
- **WHEN** 计划只启用一个标的并把其比例设置为 100%
- **THEN** 系统允许启用该计划并把全部建议定投金额分配给该标的

#### Scenario: Reject duplicate active investment target names
- **WHEN** 用户新增或修改标的后与同一计划的另一个启用标的名称重复
- **THEN** 系统 SHALL 阻止保存并提示投资标的名称必须在计划内唯一

#### Scenario: Remove an unused investment target
- **WHEN** 用户删除一个从未被月度记录引用的标的
- **THEN** 系统物理删除该标的并从计划设置中移除

#### Scenario: Remove an investment target referenced by history
- **WHEN** 用户删除一个已经被月度记录引用的标的
- **THEN** 系统 SHALL 将其归档而非物理删除，并继续保留历史名称、建议金额和实际金额

### Requirement: Validate active destination allocations
系统 SHALL 通过比例下拉、实时合计、剩余比例提示和显式联动操作帮助用户配置标的比例，并仅在计划至少有一个启用标的、所有启用标的比例非负且合计为 100% 时把计划标记为可用于月度计算。

#### Scenario: Select an allocation percentage
- **WHEN** 用户为启用标的选择比例
- **THEN** 系统 SHALL 从常用离散比例中设置该值，并立即更新总计与剩余比例

#### Scenario: Fill the remaining allocation
- **WHEN** 当前合计低于 100% 且用户对某个启用标的选择补足剩余
- **THEN** 系统 SHALL 将该标的比例调整为使启用项合计恰好 100% 的值

#### Scenario: Distribute allocations evenly
- **WHEN** 用户选择智能均分且存在至少一个启用标的
- **THEN** 系统 SHALL 稳定地把 100 个整百分比点分配给所有启用标的并使合计恰好为 100%

#### Scenario: Activate a valid plan
- **WHEN** 计划默认参数有效、至少启用一个标的且启用标的比例合计为 100%
- **THEN** 系统将计划标记为可用并开放计划内月度录入入口

#### Scenario: No investment target is active
- **WHEN** 用户停用计划的全部标的
- **THEN** 系统 SHALL 保留计划为草稿或不可计算状态，并提示至少启用一个投资标的

#### Scenario: Active allocations do not total 100 percent
- **WHEN** 启用标的比例之和不等于 100%
- **THEN** 系统 SHALL 阻止启用或计算，并显示当前合计、距离 100% 的差额与可用修正操作
