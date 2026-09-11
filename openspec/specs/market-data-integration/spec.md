# Purpose

定义基金、ETF、指数和汇率数据的提供方接入、采集口径、快照、追溯、授权及降级规则。
## Requirements

### Requirement: 市场数据必须通过可替换提供方接入
系统 SHALL 通过稳定的内部领域接口接入基金、ETF、指数和汇率数据，并 MUST 避免让业务 API 或前端依赖上游原始字段。

#### Scenario: 切换数据提供方
- **WHEN** 管理员为某类数据启用新的提供方
- **THEN** 业务 API MUST 保持相同响应契约
- **AND** 已保存的计划和月度记录 MUST 不需要迁移上游字段

#### Scenario: 部署官方 AKTools 适配器
- **WHEN** 生产环境启用 AKTools/AKShare 提供方
- **THEN** 系统 MUST 使用经过验证并固定的版本
- **AND** AKTools 服务 MUST 仅在内部网络可达

### Requirement: QDII 真实数据能力必须内建常开
系统 SHALL 始终启动 QDII 基金目录、申购状态与 ETF 行情的真实数据采集和快照读取，不得使用功能开关关闭 QDII 助手；AKTools 服务 MUST 不暴露宿主机端口，同时 SHALL 具备访问其公开上游所需的出站网络。

#### Scenario: 未提供 QDII 功能开关
- **WHEN** 应用在本地或已确认数据条款的生产部署中启动
- **THEN** 系统 SHALL 自动启动 QDII 数据刷新与助手服务
- **AND** MUST NOT 返回“QDII 助手未启用”

#### Scenario: 上游暂时不可访问
- **WHEN** AKTools 或其公开上游请求失败
- **THEN** QDII 助手 SHALL 读取最近一次成功快照，或在从未成功时显示 `--` 和真实不可用原因
- **AND** MUST NOT 把上游失败伪装为功能关闭

### Requirement: QDII 基金字段必须按已确认接口和口径采集
系统 SHALL 分别使用 QDII 目录、申购状态、基金概况、历史净值和 ETF 行情接口填充对应领域字段，并 MUST NOT 用其他含义相近的字段替代缺失值。

#### Scenario: 发现并补全 QDII 场外候选
- **WHEN** 刷新 QDII 场外基金数据
- **THEN** 适配器 SHALL 使用 `fund_open_fund_rank_em(symbol="QDII")` 发现候选
- **AND** SHALL 使用 `fund_purchase_em` 补充申购状态、下一开放日、最低购买额和渠道日累计限额
- **AND** SHALL 使用 `fund_overview_em` 提供待人工确认的跟踪标的和业绩比较基准文本

#### Scenario: 首次刷新发现新的精确映射候选
- **WHEN** QDII 目录中的新基金通过 `fund_overview_em` 精确基准验证并写入映射
- **THEN** 采集器 SHALL 在同一次刷新中写入该基金的申购状态和限额快照
- **AND** MUST NOT 因数据库时间精度或重新查询的生效时间边界把它延迟到下一次刷新
- **AND** 不在 QDII 开放式基金目录中的证券代码 MUST NOT 被误建为场外 QDII 基金

#### Scenario: 新候选概况验证较慢
- **WHEN** 已有精确映射基金和尚待验证的新候选同时出现在全市场申购数据中
- **THEN** 采集器 SHALL 先写入全部已有映射基金的最新申购快照
- **AND** SHALL 在第二遍处理新候选的概况验证与映射
- **AND** 新候选数量或概况接口延迟 MUST NOT 让已有基金继续显示旧快照

#### Scenario: QDII 排名目录临时省略已有映射基金
- **WHEN** 一个已有精确映射的场外 QDII 基金仍出现在 `fund_purchase_em`，但当天未出现在 QDII 排名目录
- **THEN** 采集器 SHALL 继续按基金代码刷新其申购状态和限额
- **AND** MUST NOT 用排名目录的临时缺席撤销或屏蔽已经确认的精确映射

#### Scenario: 默认部署获取 USD 和 HKD 规划汇率
- **WHEN** 未配置专用中国外汇交易中心端点，或该端点没有返回有效汇率
- **THEN** 采集器 SHALL 通过内网 AKTools 的 `currency_boc_safe` 获取最新同时有效的 USD 和 HKD 中间价
- **AND** SHALL 把人民币/100 外币归一化为人民币/单位外币
- **AND** SHALL 保存国家外汇管理局来源地址、观测日期和抓取时间

#### Scenario: 启动时历史序列刷新耗时较长
- **WHEN** 应用启动后的基金或 ETF 历史序列刷新缓慢
- **THEN** 采集器 SHALL 先完成 QDII 申购和 FX 快照刷新，再执行慢速历史任务
- **AND** 慢速任务 MUST NOT 占满并发槽而延迟两个助手的首屏关键数据

#### Scenario: 采集基金净值与 ETF 行情
- **WHEN** 刷新基金历史净值和场内 ETF 参考数据
- **THEN** 适配器 SHALL 使用 `fund_open_fund_info_em` 保存明确标记口径的单位净值或累计净值
- **AND** SHALL 使用 `fund_etf_spot_em` 保存 ETF 最新价、IOPV 和溢折率的可用字段
- **AND** SHALL 使用 `fund_etf_hist_em(period="daily", adjust="qfq")` 保存已映射场内 ETF 的人民币前复权收盘序列
- **AND** 当前复权接口失败时，只有 `fund_etf_hist_sina` 日收盘序列在最近一年无已报告分红且无疑似拆分跳变，才可作为明确标记的降级序列
- **AND** 任一上游字段缺失时 MUST 保持 `--`，不得从无关字段推导

#### Scenario: 采集免费 QQQ/SPY 代理比较序列
- **WHEN** 刷新免费跟踪误差代理数据
- **THEN** 适配器 SHALL 使用 `stock_us_daily` 的前复权日线采集 QQQ 和 SPY
- **AND** SHALL 使用 `currency_boc_safe` 采集国家外汇管理局每日 USD/CNY 中间价并把人民币/100 美元归一化为人民币/美元
- **AND** SHALL 保存累计净值与人民币 QQQ/SPY 代理序列的原始口径、来源和时间，不得将其改名为严格复权净值或官方总回报指数

#### Scenario: 缓存派生跟踪误差
- **WHEN** 免费代理所需的基金、QQQ/SPY 和 USD/CNY 序列刷新成功
- **THEN** 采集器 SHALL 计算一次年化跟踪误差并保存独立派生快照
- **AND** 业务 API SHALL 优先读取派生快照，不因每次打开助手而重复加载全年序列计算
- **AND** 派生快照 SHALL 持久化在现有快照仓储中，Redis 不得成为单机部署的必需依赖

#### Scenario: 历史净值序列包含缺失观测
- **WHEN** 历史累计净值响应中的个别日期明确没有净值数值，而其余有效观测仍满足样本与新鲜度要求
- **THEN** 适配器 SHALL 跳过这些无观测日期并使用其余有效观测计算
- **AND** MUST NOT 将缺失净值填充为 `0` 或生成值
- **AND** 有效共同观测日不足时仍 MUST 按缺失数据处理

### Requirement: 开源代码许可与上游数据授权必须分别控制
系统 MUST 记录所采用开源组件的名称、版本和许可证，并 MUST 为每个外部数据提供方单独记录生产使用结论。

#### Scenario: 复用官方 MIT 组件
- **WHEN** 部署或分发 AKShare/AKTools 相关实现
- **THEN** 项目 SHALL 保留其许可证声明和固定版本记录
- **AND** MUST NOT 因代码采用 MIT 许可证就把上游数据视为已获商业授权

#### Scenario: 参考未声明许可证的仓库
- **WHEN** 设计参考 QDII 限购监控仓库但无法确认其代码许可证
- **THEN** 实现 MUST 仅复用公开事实、接口行为和通用架构思想
- **AND** MUST NOT 复制该仓库的源代码、测试样本或受版权保护的表达

### Requirement: 每个外部值必须可追溯
系统 SHALL 为每个外部数据值保留提供方、来源地址、观测时间、抓取时间和新鲜度状态。

#### Scenario: 展示基金限额
- **WHEN** 助手展示某基金的申购限额
- **THEN** 响应 MUST 同时包含渠道、提供方和数据时间

#### Scenario: 展示人民币参考汇率
- **WHEN** 助手展示 USD 或 HKD 规划汇率
- **THEN** 响应 MUST 标明该值是中国外汇交易中心中间价、备用来源或用户情景值

### Requirement: 外部数据失败不得阻断核心计划流程
系统 MUST 从最近一次成功快照读取助手数据，并 SHALL 在上游不可用时降级，而不是阻断月度建议的加载或保存。

#### Scenario: 上游超时但存在旧快照
- **WHEN** 上游刷新超时且存在超过新鲜阈值的成功快照
- **THEN** 系统 SHALL 返回旧快照并标记为 `stale`
- **AND** 月度建议与记录操作 SHALL 保持可用

#### Scenario: 从未取得有效数据
- **WHEN** 某字段没有任何成功快照或有效覆盖值
- **THEN** 用户界面 MUST 显示 `--`
- **AND** 系统 MUST 不以 `0` 或生成值代替缺失数据

#### Scenario: JSONP 或数组字段契约变化
- **WHEN** 东方财富响应不再符合已固定的 JSONP 包装或数组字段位置契约
- **THEN** 刷新任务 MUST 拒绝把该响应写成新的成功快照
- **AND** SHALL 返回旧快照并标记为 `stale`，或在没有旧快照时显示 `--`

### Requirement: 标的与市场数据必须精确映射
系统 MUST 使用受管理的标的市场档案、基金代码和基准标识进行匹配，并 MUST NOT 仅根据名称模糊匹配直接生成面向用户的结果。

#### Scenario: 名称相似但没有确认映射
- **WHEN** 一个基金名称包含目标指数名称但没有生效的精确映射
- **THEN** 该基金 MUST NOT 出现在用户可执行候选中

### Requirement: 渠道限额必须保持渠道语义
系统 SHALL 以基金代码、份额类别和销售渠道为粒度保存申购状态及限额，并 MUST NOT 将单一渠道值描述为全市场统一限额。

#### Scenario: 天天基金返回日限额
- **WHEN** `fund_purchase_em` 返回某基金的日累计限额
- **THEN** 系统 MUST 在快照与领域结果中保留具体渠道和来源
- **AND** 普通用户界面的候选行 MUST NOT 展示“天天基金”等渠道名称
- **AND** 表格下方的小字说明 MUST 将限额描述为公开销售渠道观察值、仅供参考并以实际销售平台为准

#### Scenario: 用户配置其他渠道覆盖
- **WHEN** 存在有效的渠道限额覆盖
- **THEN** 系统 SHALL 使用覆盖值参与该用户的容量计算
- **AND** MUST 保留覆盖来源和生效时间

#### Scenario: 批量限额为零或与起购额矛盾
- **WHEN** 已映射基金的批量限额为零、缺失或低于起购额
- **THEN** 采集器 SHALL 在有限时间预算内核验同基金费率详情页，并校验页面基金代码
- **AND** 未开通代销 SHALL 单独标记，不得从批量“限大额”推断该渠道可执行
- **AND** 详情失败 MUST 保留未知值，不能写入猜测的成功快照

### Requirement: 凭据和内部采集服务不得暴露
系统 MUST 将 Tushare token 等凭据保存在服务端秘密配置中，并 MUST 仅在内部网络暴露 AKTools 服务。

#### Scenario: 获取前端助手数据
- **WHEN** 浏览器请求助手 API
- **THEN** 响应 MUST NOT 包含提供方 token、内部服务地址或原始错误正文
