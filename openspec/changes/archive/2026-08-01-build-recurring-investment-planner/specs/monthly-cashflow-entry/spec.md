## ADDED Requirements

### Requirement: Maintain reusable expense sources
系统 SHALL 为账号预置微信、支付宝、美团、抖音、银行卡、交通和其他等支出来源，并允许用户新增、重命名、排序和停用来源，以供该用户的各计划复用。

#### Scenario: Add a custom expense source
- **WHEN** 用户输入一个不与启用来源重名的非空名称并保存
- **THEN** 系统把该来源加入该用户之后的计划月度录入表单

#### Scenario: Disable a source used by history
- **WHEN** 用户停用一个已经出现在历史月份中的来源
- **THEN** 系统不再默认展示该来源用于新月份录入，但 SHALL 保留历史月份中的名称和金额

#### Scenario: Reject duplicate active source names
- **WHEN** 用户新增或重命名来源后与另一个启用来源名称重复
- **THEN** 系统 SHALL 阻止保存并提示来源名称必须唯一

### Requirement: Enter cashflow inside a selected plan
系统 SHALL 只允许用户从一个归属于自己的有效计划进入月度录入，并在该计划中为自然月录入一笔非负总收入和各支出来源的非负汇总金额。

#### Scenario: Enter monthly cashflow from plan details
- **WHEN** 用户打开一个有效计划的月度录入、选择月份并填写有效收入和支出
- **THEN** 系统把输入保存到该计划，并显示所有来源金额之和作为该计划该月总支出

#### Scenario: Attempt entry without a plan
- **WHEN** 用户没有选择计划而请求创建月度收支
- **THEN** 系统 SHALL 阻止操作并引导用户先创建或选择计划

#### Scenario: Attempt entry in an invalid plan
- **WHEN** 计划仍为草稿、已归档、没有启用目的地或目的地比例不合计为 100%
- **THEN** 系统 SHALL 阻止创建新月份并引导用户完成计划设置

#### Scenario: Treat blank expense amounts as zero
- **WHEN** 用户保留某个支出来源金额为空
- **THEN** 系统在合计和计算中将该来源金额视为零

#### Scenario: Reject negative or malformed amounts
- **WHEN** 用户输入负数、无法解析的金额或超过两位小数的人民币金额
- **THEN** 系统 SHALL 阻止计算和保存，并标记无效字段

### Requirement: Keep one cashflow record per plan and month
系统 SHALL 使用 \`YYYY-MM\` 标识月份，并确保同一计划的同一月份只有一份可编辑月度记录。

#### Scenario: Reopen an existing plan month
- **WHEN** 用户在同一计划中选择一个已经保存的月份
- **THEN** 系统加载该计划该月的现有输入，而不是创建重复记录

#### Scenario: Save the same month in another plan
- **WHEN** 用户在另一个计划中选择相同月份
- **THEN** 系统允许创建独立记录，且两个计划的数据互不覆盖

### Requirement: Warn about duplicate transaction counting
系统 SHALL 在计划内支出录入区域提示同一笔交易只能归入一个支出来源，并说明该月收入和支出只用于当前计划的计算。

#### Scenario: View expense entry guidance
- **WHEN** 用户进入或编辑计划月度支出表单
- **THEN** 系统展示防止来源重复计算及避免跨计划误用同一收支数据的说明

### Requirement: Reuse the prior month source layout
系统 SHALL 允许用户复制同一计划最近一个已保存月份使用的来源排列，同时不复制金额。

#### Scenario: Copy prior month layout
- **WHEN** 用户在计划的新月份选择复制上月来源列表
- **THEN** 系统按该计划上月顺序展示相同来源，并将所有新月份金额初始化为空或零

### Requirement: Scope cashflow to the authenticated user and selected plan
系统 SHALL 只在当前登录用户拥有的选定计划内创建、查询和修改月度收支记录。

#### Scenario: Load a month owned by the selected plan
- **WHEN** 用户在计划详情选择月份
- **THEN** 系统只查询该用户该计划的对应记录，即使其他计划或用户也保存了相同月份

#### Scenario: Move a month between plans
- **WHEN** 客户端尝试通过修改 plan ID 把现有月度记录移动到另一个计划
- **THEN** 系统 SHALL 拒绝操作；用户必须在目标计划创建独立月份

#### Scenario: Modify another user's expense source
- **WHEN** 用户提交另一个账号的支出来源标识进行重命名、排序或停用
- **THEN** 系统 SHALL 按来源不存在处理并保持目标来源不变
