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
系统 SHALL 允许每个已登录用户创建多个定投计划，并把计划作为投资标的、月度收支、计算、实际投入、历史和统计的一级容器。

#### Scenario: Create a plan
- **WHEN** 用户输入非空计划名称并创建计划
- **THEN** 系统保存一个归属于该用户的草稿计划，加入可编辑的初始投资标的，并导航到计划详情设置页

#### Scenario: List owned plans
- **WHEN** 已登录用户打开计划首页
- **THEN** 系统只列出该用户的计划及每个计划的状态和摘要

#### Scenario: Use the same month in different plans
- **WHEN** 同一用户的两个计划分别保存相同的 \`YYYY-MM\` 月份
- **THEN** 系统分别保存两条互不覆盖的计划月度记录

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
