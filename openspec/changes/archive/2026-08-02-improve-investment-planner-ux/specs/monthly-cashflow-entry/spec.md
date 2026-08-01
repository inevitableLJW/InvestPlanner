## MODIFIED Requirements

### Requirement: Maintain reusable expense sources
系统 SHALL 为账号预置微信、支付宝、美团、抖音、银行卡、交通和其他等支出来源，允许用户新增、重命名、排序和停用来源，并在月度录入中为每个来源显示对应的 App 或类别图标；无法识别的自定义来源 SHALL 使用一致的回退图标。

#### Scenario: Identify a common expense app
- **WHEN** 月度录入包含微信、支付宝、美团或抖音来源
- **THEN** 系统 SHALL 在名称旁显示可区分的对应 App 图标与视觉标识

#### Scenario: Identify a category expense source
- **WHEN** 月度录入包含银行卡、交通或其他来源
- **THEN** 系统 SHALL 显示对应类别图标以帮助用户快速定位

#### Scenario: Display a custom expense source
- **WHEN** 月度录入包含没有专用映射的自定义来源
- **THEN** 系统 SHALL 显示通用钱包图标并继续展示完整来源名称

#### Scenario: Add a custom expense source
- **WHEN** 用户输入一个不与启用来源重名的非空名称并保存
- **THEN** 系统把该来源加入该用户之后的计划月度录入表单

#### Scenario: Disable a source used by history
- **WHEN** 用户停用一个已经出现在历史月份中的来源
- **THEN** 系统不再默认展示该来源用于新月份录入，但 SHALL 保留历史月份中的名称、图标回退和金额

#### Scenario: Reject duplicate active source names
- **WHEN** 用户新增或重命名来源后与另一个启用来源名称重复
- **THEN** 系统 SHALL 阻止保存并提示来源名称必须唯一

### Requirement: Enter cashflow inside a selected plan
系统 SHALL 只允许用户从一个归属于自己的有效计划进入月度录入，并在该计划中为自然月录入一笔非负总收入和各支出 App 或来源的非负汇总金额，同时实时显示支出合计与预计可投资余额。

#### Scenario: Enter monthly cashflow from plan details
- **WHEN** 用户打开一个有效计划的月度录入、选择月份并填写有效收入和支出
- **THEN** 系统把输入保存到该计划，并即时显示所有来源金额之和及基于当前输入的摘要

#### Scenario: Attempt entry without a plan
- **WHEN** 用户没有选择计划而请求创建月度收支
- **THEN** 系统 SHALL 阻止操作并引导用户先创建或选择计划

#### Scenario: Attempt entry in an invalid plan
- **WHEN** 计划仍为草稿、已归档、没有启用标的或标的比例不合计为 100%
- **THEN** 系统 SHALL 阻止创建新月份并引导用户完成计划设置

#### Scenario: Treat blank expense amounts as zero
- **WHEN** 用户保留某个支出来源金额为空
- **THEN** 系统在合计和计算中将该来源金额视为零

#### Scenario: Reject negative or malformed amounts
- **WHEN** 用户输入负数、无法解析的金额或超过两位小数的人民币金额
- **THEN** 系统 SHALL 阻止计算和保存，并标记无效字段
