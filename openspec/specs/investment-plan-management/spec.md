# Purpose

定义账户认证、数据隔离、定投计划及目的地生命周期规则。

## Requirements


### Requirement: Register an account
系统 SHALL 允许未登录用户使用唯一用户名和符合密码策略的密码创建账号，并在注册事务中为新用户建立常用支出来源，但不得替用户自动创建业务计划。

#### Scenario: Register successfully
- **WHEN** 用户提交格式有效且尚未使用的用户名，以及至少 10 个字符且不超过 72 字节的密码
- **THEN** 系统创建账号、常用支出来源和登录会话，且 SHALL 只保存密码哈希

#### Scenario: Reject a duplicate username
- **WHEN** 用户提交已经注册的用户名，不区分用户名大小写
- **THEN** 系统 SHALL 拒绝注册并返回不会暴露其他账号详细信息的冲突提示

#### Scenario: Reject a weak password
- **WHEN** 用户提交不符合长度策略的密码
- **THEN** 系统 SHALL 拒绝注册并明确展示密码长度要求

### Requirement: Authenticate with a server-side session
系统 SHALL 允许用户使用用户名和密码登录，通过安全的 HttpOnly Cookie 维持可撤销且有过期时间的服务端会话，并允许用户退出。

#### Scenario: Log in successfully
- **WHEN** 用户提交正确的用户名和密码
- **THEN** 系统创建新会话、设置安全会话 Cookie，并返回当前用户的非敏感资料

#### Scenario: Reject invalid credentials generically
- **WHEN** 用户名不存在或密码错误
- **THEN** 系统 SHALL 返回相同的通用认证失败响应，且不得说明用户名是否已注册

#### Scenario: Restore an active session
- **WHEN** 用户刷新应用且浏览器携带未过期、未撤销的会话 Cookie
- **THEN** 系统恢复登录状态并加载该用户的计划列表

#### Scenario: Log out
- **WHEN** 已登录用户执行退出
- **THEN** 系统撤销当前服务端会话、清除 Cookie，并拒绝该会话之后的受保护请求

### Requirement: Protect and isolate user data
系统 SHALL 要求计划、目的地、支出来源、月度记录、实际投入和统计接口经过认证，并只返回或修改当前登录用户拥有的数据。

#### Scenario: Request protected data without a session
- **WHEN** 未登录客户端请求任何业务数据接口
- **THEN** 系统 SHALL 返回 401 且不泄露业务数据

#### Scenario: Attempt to access another user's plan
- **WHEN** 已登录用户使用另一个用户的计划标识请求计划或任意子资源
- **THEN** 系统 SHALL 按计划不存在处理，并保持目标计划及其子资源不变

### Requirement: Create and list investment plans
系统 SHALL 允许每个已登录用户创建多个定投计划，并把计划作为目的地、月度收支、计算、实际投入、历史和统计的一级容器。

#### Scenario: Create a plan
- **WHEN** 用户输入非空计划名称并创建计划
- **THEN** 系统保存一个归属于该用户的草稿计划，加入支付宝基金、A股、港美股、现金四个可编辑默认目的地，并导航到计划详情

#### Scenario: List owned plans
- **WHEN** 已登录用户打开计划首页
- **THEN** 系统只列出该用户的计划及每个计划的状态和摘要

#### Scenario: Use the same month in different plans
- **WHEN** 同一用户的两个计划分别保存相同的 \`YYYY-MM\` 月份
- **THEN** 系统分别保存两条互不覆盖的计划月度记录

### Requirement: Configure plan defaults
系统 SHALL 允许用户为每个计划设置默认定投比例、非负预留资金和正数取整单位，并提供常用比例快捷选项及 0% 到 100% 的自定义比例。

#### Scenario: Save plan defaults
- **WHEN** 用户提交有效的默认比例、预留资金和取整单位
- **THEN** 系统保存这些参数，并用它们初始化该计划之后的新月份

#### Scenario: Enter a custom default rate
- **WHEN** 用户输入范围有效但不属于快捷项的自定义比例
- **THEN** 系统 SHALL 精确保留该比例

#### Scenario: Reject invalid plan defaults
- **WHEN** 用户提交负预留资金、非正取整单位或超出 0% 到 100% 的比例
- **THEN** 系统 SHALL 阻止计划启用并标记无效字段

### Requirement: Manage configurable contribution destinations
系统 SHALL 允许用户在每个计划中新增、重命名、排序、启用和停用定投目的地；支付宝基金、A股、港美股、现金仅为新计划默认模板，不是强制项。

#### Scenario: Add a custom destination
- **WHEN** 用户在计划中添加一个名称非空且不与该计划其他启用目的地重名的目的地
- **THEN** 系统保存该目的地并允许用户为其设置分配比例

#### Scenario: Disable default destinations
- **WHEN** 用户停用支付宝基金、A股、港美股或现金中的任意默认目的地
- **THEN** 系统不再把该目的地用于新月份计算，且 SHALL 不要求用户重新启用它

#### Scenario: Use only one destination
- **WHEN** 计划只启用一个目的地并把其比例设置为 100%
- **THEN** 系统允许启用该计划并把全部建议定投金额分配给该目的地

#### Scenario: Reject duplicate active destination names
- **WHEN** 用户新增或重命名目的地后与同一计划的另一个启用目的地名称重复
- **THEN** 系统 SHALL 阻止保存并提示目的地名称必须在计划内唯一

#### Scenario: Remove an unused destination
- **WHEN** 用户删除一个从未被月度记录引用的目的地
- **THEN** 系统物理删除该目的地并从计划设置中移除

#### Scenario: Remove a destination referenced by history
- **WHEN** 用户删除一个已经被月度记录引用的目的地
- **THEN** 系统 SHALL 将其归档而非物理删除，并继续保留历史名称、建议金额和实际金额

### Requirement: Validate active destination allocations
系统 SHALL 仅在计划至少有一个启用目的地、所有启用目的地比例非负且合计为 100% 时把计划标记为可用于月度计算。

#### Scenario: Activate a valid plan
- **WHEN** 计划默认参数有效、至少启用一个目的地且启用目的地比例合计为 100%
- **THEN** 系统将计划标记为可用并开放计划内月度录入入口

#### Scenario: No destination is active
- **WHEN** 用户停用计划的全部目的地
- **THEN** 系统 SHALL 保留计划为草稿或不可计算状态，并提示至少启用一个目的地

#### Scenario: Active allocations do not total 100 percent
- **WHEN** 启用目的地比例之和不等于 100%
- **THEN** 系统 SHALL 阻止启用或计算，并显示当前合计与所需合计

### Requirement: Archive a plan without destroying history
系统 SHALL 允许用户归档计划；有月度历史的计划不得被物理删除。

#### Scenario: Archive a plan
- **WHEN** 用户确认归档一个计划
- **THEN** 系统从默认计划列表中隐藏该计划、阻止新增月份，并保留其历史和统计供查看

### Requirement: Persist plans in MySQL
系统 SHALL 通过 Gin API 和 GORM 将每个用户的计划及目的地持久保存到 MySQL，并在之后的已认证会话中恢复。

#### Scenario: Reload plans
- **WHEN** 用户创建或修改计划后刷新应用或重新登录
- **THEN** 系统从后端加载该账号最近一次成功保存的计划列表、设置和目的地

#### Scenario: Database persistence fails
- **WHEN** 后端无法提交计划或目的地保存事务
- **THEN** 系统 SHALL 返回可重试错误、保持原数据不变，并且前端不得显示保存成功
