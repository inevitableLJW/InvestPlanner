# Purpose

定义定投计划器的响应式界面、可访问反馈、流程导航、投资标的排序及外观设置规则。
## Requirements
### Requirement: Present a coherent responsive planner interface
系统 SHALL 使用一致的视觉层级、间距、控件状态和响应式布局呈现计划列表、计划设置、月度录入、结果与历史，并在桌面和窄屏设备上保持主要内容及操作可读可用。

#### Scenario: Use the planner on a narrow screen
- **WHEN** 用户在窄屏设备打开计划设置或月度录入
- **THEN** 系统 SHALL 将多列内容重排为单列或可滚动布局，且不得遮挡主要动作、字段标签或校验信息

#### Scenario: Scan a complex form
- **WHEN** 用户打开包含多个配置项的设置或月度录入页面
- **THEN** 系统 SHALL 使用标题、说明、摘要和分区卡片明确当前状态、输入顺序与主要动作

### Requirement: Provide accessible feedback and motion
系统 SHALL 为加载、保存、无数据、校验失败和成功状态提供就近、可被辅助技术识别的反馈，并仅使用不阻碍操作的轻量过渡与进入动效。

#### Scenario: Save a form
- **WHEN** 用户提交设置或月度记录
- **THEN** 系统 SHALL 禁用重复提交、展示进行中状态，并在完成后明确展示成功或失败结果

#### Scenario: Prefer reduced motion
- **WHEN** 操作系统启用了减少动态效果偏好
- **THEN** 系统 SHALL 关闭非必要位移和进入动画，同时保留状态变化的可见反馈

### Requirement: Keep primary actions and next steps clear
系统 SHALL 在计划配置和月度录入流程中突出与当前意图匹配的主要动作，并根据当前计划状态说明用户可执行的下一步。计划配置页 SHALL 同时提供语义明确的“保存草稿”和“发布计划”动作，不得使用一个含义模糊的保存动作隐式开始计划。

#### Scenario: Plan configuration is incomplete
- **WHEN** 用户尚未配置至少一个启用标的或启用标的比例未达到 100%
- **THEN** 系统 SHALL 展示差额和修正入口、允许保存草稿，并阻止发布计划

#### Scenario: Plan configuration is ready
- **WHEN** 启用标的名称有效且比例合计为 100%
- **THEN** 系统 SHALL 明确标记配置已就绪，并允许用户选择保存草稿或发布计划

#### Scenario: Choose to save rather than publish
- **WHEN** 用户点击保存草稿
- **THEN** 系统 SHALL 在操作期间阻止重复提交，并在成功后明确说明计划仍处于草稿且尚未开始

#### Scenario: Choose to publish
- **WHEN** 用户点击发布计划
- **THEN** 系统 SHALL 在操作期间阻止重复提交，并在成功后明确说明计划已开始且可进行月度录入

### Requirement: Organize investment targets predictably
系统 SHALL 支持用户按投资标的比例从高到低排序，也 SHALL 支持通过拖拽手动调整顺序，并在保存时提交与视觉顺序一致的 `sortOrder`。系统 SHALL 保留可通过键盘操作的上移、下移控件作为拖拽的替代方式。

#### Scenario: Sort targets by allocation
- **WHEN** 用户在投资标的设置中选择按比例排序
- **THEN** 系统 SHALL 将启用标的按分配比例从高到低排列，并稳定保留比例相同项的原有先后顺序

#### Scenario: Drag a target to a new position
- **WHEN** 用户将一个投资标的拖到另一个标的的位置
- **THEN** 系统 SHALL 立即更新显示顺序和各项 `sortOrder`，且保存后的概览与后续月度计算使用该顺序

### Requirement: Complete configuration with clear navigation and appearance controls
系统 SHALL 使用稳定的计划 ID 路径在概览、设置、月度记录、历史记录和统计 Tab 之间切换。保存草稿或发布计划成功后系统 SHALL 展示与该动作结果一致的确认反馈；用户确认后 SHALL 返回该计划概览。应用 SHALL 提供至少三套可切换皮肤并记住用户选择。

#### Scenario: Confirm a successful settings save
- **WHEN** 用户成功保存计划草稿
- **THEN** 系统 SHALL 展示草稿已保存且计划尚未开始的确认反馈；用户确认后 SHALL 返回该计划概览

#### Scenario: Confirm a successful publish
- **WHEN** 用户成功发布计划
- **THEN** 系统 SHALL 展示计划已发布且可以开始月度记录的确认反馈；用户确认后 SHALL 返回该计划概览

#### Scenario: Switch a plan tab
- **WHEN** 用户从任意计划内页面点击任一 Tab
- **THEN** 系统 SHALL 打开同一计划对应的页面，且 Tab 的激活状态与当前页面一致

#### Scenario: Choose a skin
- **WHEN** 用户在应用顶栏选择另一套皮肤
- **THEN** 系统 SHALL 立即应用所选色彩 token，并在下次打开应用时恢复该选择

#### Scenario: Preview and choose a skin
- **WHEN** 用户打开皮肤下拉菜单
- **THEN** 系统 SHALL 在每个皮肤名称旁显示对应的颜色预览块

#### Scenario: Choose display mode
- **WHEN** 用户切换亮色或暗色模式
- **THEN** 系统 SHALL 立即以所选模式呈现所有页面，并在下次打开应用时恢复该选择

### Requirement: Confirm permanent deletion of an unstarted plan
系统 SHALL 在计划列表中只为服务端判定可永久删除的未开始计划展示“删除草稿”危险操作，并在执行前明确说明该操作不可恢复。删除期间系统 SHALL 阻止重复提交，成功后从列表移除该计划，失败时 SHALL 保留当前列表并展示原因。

#### Scenario: Confirm deletion of an eligible draft
- **WHEN** 用户在计划列表中选择可删除草稿计划的“删除草稿”
- **THEN** 系统 SHALL 展示包含计划名称和不可恢复说明的确认对话框，只有用户明确确认后才提交删除请求

#### Scenario: Cancel draft deletion
- **WHEN** 用户在永久删除确认对话框中选择取消
- **THEN** 系统 SHALL 关闭对话框并保持计划及当前编辑内容不变

#### Scenario: Complete draft deletion
- **WHEN** 服务端成功永久删除未开始计划
- **THEN** 系统 SHALL 返回计划列表、移除已删除计划且不得再导航到其详情页

#### Scenario: Reject stale deletion eligibility
- **WHEN** 删除请求因计划已发布、已有月度记录或版本冲突而失败
- **THEN** 系统 SHALL 保留当前页面、明确说明计划未被删除并提示用户刷新状态

### Requirement: Evaluate simple income expressions

系统 SHALL 允许用户在月度“上月总收入”字段输入仅含非负金额、加号和减号的简单算式，并在预览与提交时使用计算结果；系统 MUST 不执行任意脚本或保存原始算式文本。

#### Scenario: Preview an income expression
- **WHEN** 用户输入 `20000+500-100`
- **THEN** 系统将收入按 20,400 元计算，在输入框下方显示当前合计，并用同一个金额减去支出和预留显示精确的预计可投入金额，不按 100 元取整

#### Scenario: Save an evaluated income expression
- **WHEN** 用户输入有效收入算式并生成或重新计算本月投资建议
- **THEN** 系统 SHALL 将算式计算后的金额作为 `incomeCents` 提交和保存，本月投资建议中的本月收入不得变为 0

#### Scenario: Reject an invalid income expression
- **WHEN** 用户输入包含其他字符、格式不完整或计算结果为负的算式
- **THEN** 系统 SHALL 标记该字段无效、说明仅支持加减法且不得提交月度记录

### Requirement: 月度记录页必须动态形成海外标的视觉组
系统 SHALL 只使用当前计划中实际存在的海外直投标的形成一个视觉组，并 MUST 保持每个标的与原计划数据的一一对应关系。

#### Scenario: 原计划只有两个美股标的
- **WHEN** 用户打开月度记录页
- **THEN** 海外视觉组 MUST 只包含这两个美股标的
- **AND** MUST NOT 渲染参考图中额外的港股或美股示例行

#### Scenario: 计划同时包含本地、QDII 和海外直投标的
- **WHEN** 页面渲染标的列表
- **THEN** 非海外直投标的 SHALL 保持原有行语义
- **AND** 海外直投标的 SHALL 归入单一可识别的视觉组

#### Scenario: 海外组展示实际标的统计
- **WHEN** 海外组包含美股直投或港股直投标的
- **THEN** 组头 SHALL 只显示实际存在市场的标的数量
- **AND** 组底 SHALL 显示海外直投标的总数
- **AND** MUST NOT 显示不存在市场的零数量占位

### Requirement: 换汇预测入口必须属于海外组
年度换汇预测入口 SHALL 放置在海外视觉组右下角，而 MUST NOT 挂在任意单个标的行后面。

#### Scenario: 海外组包含多个标的
- **WHEN** 页面显示可换汇预测的海外组
- **THEN** 页面 SHALL 只显示一个组级入口
- **AND** 点击入口 SHALL 打开覆盖整个海外组的年度换汇预测

#### Scenario: 没有换汇相关标的
- **WHEN** 年度换汇预测返回空币种集合
- **THEN** 页面 MUST 不显示组级入口

#### Scenario: 有真实海外直投但汇率快照暂不可用
- **WHEN** 海外组至少包含一个需个人换汇的真实直投标的
- **THEN** 页面 SHALL 在组右下角显示年度换汇预测入口
- **AND** 入口可见性 MUST NOT 依赖汇率快照是否成功
- **AND** 打开后缺失的汇率与外币金额 MUST 显示 `--` 及原因

### Requirement: QDII 助手入口必须保持标的级语义
QDII 基金助手入口 SHALL 显示在对应 QDII 标的行内，并 MUST 与组级换汇入口在视觉和文案上可区分。

#### Scenario: 同一页面同时存在 QDII 和美股直投
- **WHEN** 用户查看月度记录
- **THEN** QDII 行 SHALL 显示该标的的基金助手入口
- **AND** 美股组右下角 SHALL 显示独立的年度换汇预测入口

#### Scenario: QDII 标的已确认本月执行方案
- **WHEN** 月度记录加载到该 QDII 标的的确认摘要
- **THEN** 页面 SHALL 在该标的助手入口下方以紧凑分区卡片就地展示已选场外基金和场内承接方案
- **AND** 场外区域 SHALL 按基金展示持久化的每日投入与本月合计
- **AND** 场内区域 SHALL 同时展示已选 ETF 与持久化的场内合计
- **AND** MUST NOT 把一个 QDII 标的的方案展示到另一个标的行

### Requirement: 系统预设标的必须内置市场档案
系统 SHALL 为常用列表中的预设标的内置执行方式、结算币种、基准标识和换汇相关性，并 MUST NOT 要求用户为这些已有标的选择执行方式；只有列表外的自定义标的 SHALL 显示执行方式配置。

#### Scenario: 编辑已有预设标的
- **WHEN** 用户在计划设置中查看现金、境内基金、QDII、港股或美股预设标的
- **THEN** 页面 MUST 不显示执行方式选择器
- **AND** 保存时服务端 MUST 使用该预设标的的系统档案，不接受客户端覆盖

#### Scenario: 添加自定义标的
- **WHEN** 用户选择“自定义标的”并填写列表外名称
- **THEN** 页面 SHALL 显示执行方式选择器
- **AND** 系统 SHALL 根据所选执行方式自动设置兼容的结算币种和换汇相关性

#### Scenario: 从自定义切换回预设标的
- **WHEN** 用户把一行自定义标的改成常用列表中的预设标的
- **THEN** 系统 MUST 立即替换为该预设标的的内置市场档案
- **AND** MUST NOT 沿用此前自定义标的的执行方式或基准

### Requirement: 参考图只约束结构而不约束示例数据
实现 SHALL 使用 `assets/qdii-assistant-reference.png` 与 `assets/fx-forecast-layout-reference.png` 作为布局参考，并 MUST 以规范和真实 API 数据作为标的数量、名称、金额和币种的唯一依据。

#### Scenario: 参考图与实际计划不一致
- **WHEN** 参考图含有实际计划不存在的标的或金额
- **THEN** 实现 MUST 忽略这些示例值
- **AND** 只渲染实际计划内容

### Requirement: 助手数据状态必须清晰且不遮挡主流程
助手 SHALL 区分加载中、新鲜、陈旧和不可用状态，并 SHALL 允许用户关闭侧栏后继续填写月度实际投入。

#### Scenario: 市场数据不可用
- **WHEN** 用户打开助手但没有有效数据
- **THEN** 侧栏 SHALL 显示不可用说明与 `--`
- **AND** 月度记录表单 SHALL 保持可交互
