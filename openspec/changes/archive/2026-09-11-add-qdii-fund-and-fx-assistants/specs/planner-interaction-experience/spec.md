## ADDED Requirements

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
