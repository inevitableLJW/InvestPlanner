## MODIFIED Requirements

### Requirement: Configure plan defaults
系统 SHALL 允许用户为每个计划设置默认定投比例、非负预留资金和正数取整单位，并提供常用比例快捷选项及 0% 到 100% 的自定义比例。系统 SHALL 将变更生效后新建计划的取整单位初始化为 100 元，且不得用该默认值覆盖已有计划保存的取整单位。

#### Scenario: Create a plan with the default rounding unit
- **WHEN** 用户创建新计划
- **THEN** 系统将该计划的取整单位初始化为 100 元，并在设置页展示该值

#### Scenario: Save plan defaults
- **WHEN** 用户提交有效的默认比例、预留资金和取整单位
- **THEN** 系统保存这些参数，并用它们初始化该计划之后的新月份

#### Scenario: Enter a custom default rate
- **WHEN** 用户输入范围有效但不属于快捷项的自定义比例
- **THEN** 系统 SHALL 精确保留该比例

#### Scenario: Change the rounding unit
- **WHEN** 用户把默认的 100 元取整单位修改为另一个有效正数金额并保存
- **THEN** 系统 SHALL 精确保留用户设置的取整单位，并将其用于发布后新建月份的建议金额计算

#### Scenario: Reject invalid plan defaults
- **WHEN** 用户提交负预留资金、非正取整单位或超出 0% 到 100% 的比例
- **THEN** 系统 SHALL 阻止计划发布并标记无效字段

### Requirement: Validate active destination allocations
系统 SHALL 通过比例下拉、实时合计、剩余比例提示和显式联动操作帮助用户配置标的比例，并仅在计划至少有一个启用标的、所有启用标的比例非负且合计为 100% 时允许发布计划并用于月度计算。保存草稿 SHALL 不要求分配配置已经达到可发布条件。

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
- **WHEN** 用户选择发布，且计划默认参数有效、至少启用一个标的、启用标的名称有效且比例合计为 100%
- **THEN** 系统将计划标记为运行中并开放计划内月度录入入口

#### Scenario: Save an incomplete plan as a draft
- **WHEN** 用户选择保存草稿，而计划没有启用标的或启用比例不合计为 100%
- **THEN** 系统 SHALL 保存当前有效字段和标的配置、保持计划为草稿，且不开放月度计算

#### Scenario: No investment target is active
- **WHEN** 用户尝试发布已停用全部标的的计划
- **THEN** 系统 SHALL 保留计划为草稿或不可计算状态，并提示至少启用一个投资标的

#### Scenario: Active allocations do not total 100 percent
- **WHEN** 用户尝试发布且启用标的比例之和不等于 100%
- **THEN** 系统 SHALL 阻止发布或计算，并显示当前合计、距离 100% 的差额与可用修正操作

## ADDED Requirements

### Requirement: Save drafts and publish plans explicitly
系统 SHALL 将计划设置写入请求解释为显式的“保存草稿”或“发布计划”动作，而不是信任客户端直接指定持久化状态；保存草稿 SHALL 将计划置为草稿，发布 SHALL 在同一事务中保存当前设置、完成全部发布校验并将计划置为运行中。只有运行中计划 SHALL 被视为已开始并允许创建月度记录。

#### Scenario: Save valid settings without starting the plan
- **WHEN** 用户对一份尚未发布但配置完整的计划点击保存草稿
- **THEN** 系统保存当前设置并保持计划为草稿，不开放月度录入

#### Scenario: Publish current settings atomically
- **WHEN** 用户对配置完整的草稿点击发布
- **THEN** 系统在一个事务中保存当前设置并把计划置为运行中，随后开放月度录入

#### Scenario: Draft a previously active plan
- **WHEN** 用户编辑一份运行中计划并点击保存草稿
- **THEN** 系统保存更改、将计划置为草稿并暂停新增或重算月份，历史月份仍可查看

#### Scenario: Reject an invalid publish without losing edits
- **WHEN** 用户点击发布但当前设置未通过发布校验
- **THEN** 系统不把计划标记为运行中，前端保留当前编辑内容并展示需要修正的字段

### Requirement: Delete an unstarted draft plan permanently
系统 SHALL 允许计划所有者永久删除当前状态为草稿且没有任何月度记录的未开始计划，并 SHALL 在同一事务中删除该计划及其投资标的。系统 MUST 在执行删除时根据当前持久化状态、月度记录和版本号重新判断资格，不得仅信任客户端提供的状态或删除资格。

#### Scenario: Delete an unstarted plan
- **WHEN** 用户确认删除自己当前为草稿、没有月度记录且版本匹配的计划
- **THEN** 系统永久删除该计划及其投资标的，之后的列表和详情请求均不再返回该计划

#### Scenario: Reject deletion of a running or archived plan
- **WHEN** 用户尝试永久删除当前为运行中或已归档的计划
- **THEN** 系统 SHALL 拒绝永久删除并保持计划、投资标的和历史数据不变

#### Scenario: Reject deletion of a draft with history
- **WHEN** 用户尝试永久删除当前为草稿但已经存在至少一条月度记录的计划
- **THEN** 系统 SHALL 拒绝永久删除并提示该计划必须通过归档保留历史

#### Scenario: Deletion eligibility changes concurrently
- **WHEN** 用户确认删除后计划版本发生变化、计划被发布或产生了月度记录
- **THEN** 系统 SHALL 以冲突响应拒绝删除且不得留下部分删除的数据

#### Scenario: Attempt to delete another user's draft
- **WHEN** 用户提交另一个账户所拥有计划的永久删除请求
- **THEN** 系统 SHALL 按计划不存在处理并保持目标计划不变
