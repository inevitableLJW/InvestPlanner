## MODIFIED Requirements

### Requirement: Allocate across any active destinations
系统 SHALL 按计划中当前启用投资标的的比例分配建议定投总额，不限制标的数量，并保证所有最终金额非负且合计严格等于建议总额；面向用户的配置、结果和说明 SHALL 统一使用“投资标的”术语。

#### Scenario: Allocate to a subset of common investment targets
- **WHEN** 计划只启用常用列表中的部分投资标的且其比例合计为 100%
- **THEN** 系统只为这些启用标的生成建议金额，不为已停用标的生成新建议

#### Scenario: Allocate to a custom investment target
- **WHEN** 计划包含用户新增且已启用的自定义标的
- **THEN** 系统按照该标的比例参与计算，行为与常用标的一致

#### Scenario: Allocate to one investment target
- **WHEN** 计划只有一个比例为 100% 的启用标的
- **THEN** 系统把全部建议总额分配给该标的

#### Scenario: Allocate a rounding remainder deterministically
- **WHEN** 任意数量标的向下取整后仍有完整取整单位未分配
- **THEN** 系统按原始分项小数余数从大到小分配剩余单位，并用标的排序及稳定 ID 处理相同余数
