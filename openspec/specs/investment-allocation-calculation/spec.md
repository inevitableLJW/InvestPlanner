# Purpose

定义月度可定投金额、目的地分配、取整、快照和服务端权威计算规则。
## Requirements
### Requirement: Calculate the investable base within a plan
系统 SHALL 在选定计划中汇总该月支出，并按 \`max(月收入 - 总支出 - 预留资金, 0)\` 计算可定投基数，所有金额计算使用整数分。

#### Scenario: Positive investable base
- **WHEN** 计划月收入大于总支出与预留资金之和
- **THEN** 系统把三者相减的正数结果作为可定投基数，并展示收入、总支出、预留资金和计算过程

#### Scenario: Insufficient monthly surplus
- **WHEN** 计划月收入小于或等于总支出与预留资金之和
- **THEN** 系统将可定投基数设为零，且 SHALL 提示本月不建议定投

### Requirement: Apply a plan-scoped monthly contribution rate
系统 SHALL 用选定计划的默认比例初始化月度定投比例，允许用户为当前月份选择快捷比例或输入 0% 到 100% 的自定义比例，且月度覆盖不修改计划默认值。

#### Scenario: Use the selected plan default rate
- **WHEN** 用户打开该计划尚未保存的月份且没有修改定投比例
- **THEN** 系统使用该计划的默认定投比例计算建议金额

#### Scenario: Override the rate for one plan month
- **WHEN** 用户为当前计划月份选择或输入另一个有效比例
- **THEN** 系统只用覆盖比例计算该计划该月，并在该计划下一个新月份继续使用计划默认比例

### Requirement: Round the recommended total safely
系统 SHALL 将可定投基数乘以当月比例得到原始建议金额，并按计划取整单位向下取整，保证建议定投总额不超过可定投基数。

#### Scenario: Round down to the configured unit
- **WHEN** 原始建议金额不是取整单位的整数倍
- **THEN** 系统向下取整到最近的完整单位并展示取整后的建议总额

#### Scenario: Result is below one rounding unit
- **WHEN** 原始建议金额小于一个取整单位
- **THEN** 系统将建议定投总额设为零并说明取整影响

### Requirement: Allocate across any active destinations
系统 SHALL 按计划中当前启用投资标的的比例分配建议定投总额，不限制标的数量，并保证所有最终金额非负且合计严格等于建议总额；面向用户的配置、结果和说明 SHALL 统一使用“投资标的”术语。当且仅当存在一个启用且名称为“现金”的标的时，系统 SHALL 保留精确建议总额，将非现金标的的理论分项金额按计划取整单位向下取整，并让现金获得精确建议总额减去全部非现金分项后的余额，从而承接其配置份额、全部分项取整尾差和建议总额零头。未启用现金时，系统 SHALL 先将建议总额按计划取整单位向下取整，再按稳定规则分配。

#### Scenario: Allocate to a subset of common investment targets
- **WHEN** 计划只启用常用列表中的部分投资标的且其比例合计为 100%
- **THEN** 系统只为这些启用标的生成建议金额，不为已停用标的生成新建议

#### Scenario: Allocate to a custom investment target
- **WHEN** 计划包含用户新增且已启用的自定义标的
- **THEN** 系统按照该标的比例参与计算，行为与常用标的一致

#### Scenario: Allocate to one investment target
- **WHEN** 计划只有一个比例为 100% 的启用标的
- **THEN** 系统把全部建议总额分配给该标的

#### Scenario: Put allocation remainder into cash
- **WHEN** 启用标的包含“现金”，且建议总额或其他启用标的按比例计算后产生不足一个取整单位的零头
- **THEN** 系统将所有非现金标的向下取整，并把精确建议总额与这些分项之和的差额全部分配给现金，使最终合计严格等于精确建议总额

#### Scenario: Cash is the only active target
- **WHEN** “现金”是唯一启用且比例为 100% 的标的
- **THEN** 系统把全部建议总额分配给现金

#### Scenario: Allocate a rounding remainder deterministically
- **WHEN** 计划未启用“现金”，且任意数量标的向下取整后仍有完整取整单位未分配
- **THEN** 系统按原始分项小数余数从大到小分配剩余单位，并用标的排序及稳定 ID 处理相同余数

### Requirement: Validate plan and calculation prerequisites
系统 SHALL 只在计划归属有效、计划未归档、至少启用一个目的地、启用目的地比例合计为 100%，且所有月度输入有效时执行并保存计算。

#### Scenario: Block calculation with invalid destinations
- **WHEN** 计划没有启用目的地、启用比例不合计为 100% 或任一比例为负
- **THEN** 系统 SHALL 阻止计算、保留用户输入，并列出需要修正的计划设置

#### Scenario: Block calculation for an archived plan
- **WHEN** 用户尝试为已归档计划创建新月份
- **THEN** 系统 SHALL 阻止计算，同时允许只读查看历史月份

### Requirement: Save an explainable destination snapshot
系统 SHALL 为每个成功计算的计划月份保存输入、来源金额、计划参数、当月覆盖比例、计算中间值，以及该月全部目的地的 ID、名称、排序、比例、建议金额和实际金额快照。

#### Scenario: Change destinations after a month is calculated
- **WHEN** 用户新增、重命名、排序、停用或归档当前计划目的地后查看历史月份
- **THEN** 系统继续展示该月快照中的目的地结构与结果，不用当前设置静默改写历史

#### Scenario: Recalculate an existing plan month
- **WHEN** 用户编辑历史月份并确认重新计算
- **THEN** 系统使用该月已保存的目的地分配快照重新计算建议金额，并保留各快照目的地的实际投入

### Requirement: Calculate authoritatively on the server
系统 SHALL 在 Gin 后端使用已验证的计划归属、计划设置和月度输入重新计算所有中间值及建议金额，不得信任客户端提交的派生结果。

#### Scenario: Client submits manipulated calculation results
- **WHEN** 客户端提交的建议总额、目的地分项或执行状态与后端公式结果不一致
- **THEN** 系统忽略客户端派生值，使用后端结果响应并持久化

#### Scenario: Calculation transaction fails
- **WHEN** 后端计算成功但无法完整提交月度记录、支出明细和全部目的地明细
- **THEN** 系统 SHALL 回滚整个事务并返回可重试错误，不得留下部分快照
