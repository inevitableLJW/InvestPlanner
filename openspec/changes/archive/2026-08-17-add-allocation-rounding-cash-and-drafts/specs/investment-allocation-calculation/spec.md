## MODIFIED Requirements

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
