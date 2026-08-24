# QDII 基金助手与年度换汇预测

## 部署

AKTools 使用 [`deploy/aktools.Dockerfile`](../deploy/aktools.Dockerfile) 构建，固定为 AKTools `0.0.91`、AKShare `1.18.64`。GitHub Actions 会与 API、Web 一起发布 `linux/amd64` 和 `linux/arm64` 的 `ghcr.io/inevitableljw/invest-planner-aktools` 镜像。Compose 没有为 AKTools 声明 `ports`；它通过 `backend` 内部网络只向 API 提供服务，并通过独立的 `egress` 网络访问公开上游，宿主机和浏览器不能直接访问其端口。

TrueNAS 模板默认跟踪三个镜像的 `latest` 标签：`main` 分支镜像发布完成后，在应用页更新或重新保存配置即可拉取新镜像，不再需要自行构建或导入 AKTools。发布标签和提交构建还会生成 `v*`、`sha-*` 镜像标签；生产环境如需完全固定或回滚，应把 API、Web、AKTools 三个镜像统一替换为同一 `v*` 或对应 `sha-*` 标签。

首次启用前：

1. 阅读 [`market-data.md`](market-data.md)，确认每个上游的生产使用条款。
2. 为真实计划标的维护执行方式、结算币种和精确基准 ID；旧标的默认 `unknown`。
3. 由管理员维护 `fund_shares` 与 `benchmark_mappings` 的精确映射。名称相似不构成映射。
4. 启动后核对首次刷新日志、快照来源和单位；QDII 采集与助手为内建常开能力。

```env
AKTOOLS_BASE_URL=http://aktools:8080
TUSHARE_TOKEN=
CHINAMONEY_URL=
MARKET_DATA_PRODUCTION_APPROVED=true
FX_ASSISTANT_ENABLED=true
```

`MARKET_DATA_PRODUCTION_APPROVED` 是硬门禁：QDII 数据始终启用，因此生产环境没有部署方的显式条款确认时 API 拒绝启动。项目不再读取 `MARKET_DATA_ENABLED` 或 `QDII_ASSISTANT_ENABLED`，把它们设为 `false` 也不会关闭 QDII。`TUSHARE_TOKEN` 只在服务端读取；未配置时不会请求 Tushare，系统改用明确标记的免费 QQQ/SPY 代理口径。`CHINAMONEY_URL` 必须是部署方审查过的中国外汇交易中心数据端点，返回适配器契约中的原始单位、币种、观测日期和值；为空时不抓取，允许使用明确标记的 Tushare 备用快照。

默认刷新频率为 6 小时；ETF、申购限制、汇率的新鲜阈值分别为 15 分钟、12 小时、36 小时。采集请求 5 秒超时，限制并发；刷新失败不写入“成功”快照，业务 API 继续读取最近一次成功值并标记陈旧。

QQQ/SPY 代理跟踪误差在市场数据刷新时计算一次，并持久化为独立的派生快照；打开助手优先读取这个小型结果，不会每次重新加载全年基金、ETF 和汇率序列。当前单机/TrueNAS 部署因此不需要 Redis；将来只有在 API 多实例和高并发场景下才需要考虑 Redis 二级缓存。

回滚 QDII 数据能力需要回退应用版本或停止 AKTools；不存在运行时关闭 QDII 的功能开关。兼容回滚可保留市场档案和快照表；必须物理删除时，备份后在维护窗口执行 `server/migrations/006_market_data.down.sql`。它是破坏性回滚，不会自动运行。

## 产品语义

- 常用列表中的系统预设标的自带固定市场档案，计划设置页不显示执行方式选择器；只有列表外的自定义标的允许用户选择执行方式。服务端会强制预设档案，历史数据只回填已确认的预设名称，其他名称继续保持 `unknown`。
- QDII 数据采集与助手始终开启，只服务计划真实存在、`execution_mode=qdii` 且有精确基准映射的标的。场外限额明确标注渠道；暂停申购的新增容量是真实 `0`，未知限额是 `--`。
- 年化跟踪误差 = 最近一年共同交易日收益差的样本标准差 × `sqrt(252)`。严格口径优先使用基金复权净值和兼容总回报基准；未配置 Tushare 时，免费代理口径用基金累计净值分别对比经每日 USD/CNY 中间价换算为人民币的 QQQ（纳斯达克 100）或 SPY（标普 500）前复权序列。页面会写明“相对 QQQ/SPY · 免费代理口径”，不会把累计净值冒充复权净值，也不会把代理结果称为指数商官方口径。共同观测日少于 120、映射缺失、序列缺失或数据陈旧时显示 `--`。
- 年度换汇预测复用服务端月度分配算法和整数分精度，只纳入 `direct_us`/`direct_hk` 且 `fx_quota_relevant=true` 的真实标的。QDII、港股通、本地和未分类标的全部排除。
- 币种集合是动态的：没有美股直投就不返回 USD；没有港股直投就不返回 HKD；两者都没有时页面不显示入口。缺汇率时人民币需求仍显示，外币和 USD 等值显示 `--`。
- 未来月份默认把最新中间价当作静态规划假设，也可用用户情景汇率覆盖。它不是银行成交价。
- 50,000 USD 只显示为“年度便利化额度参考”。它不是绝对购汇上限，也不代表或构成境外证券投资许可。
- `--` 始终表示未知、缺失、口径不足或不可用；真实零值显示 `0`，二者不能互换。

助手是只读参考功能。关闭或断开 AKTools、Tushare 和汇率端点后，月度建议加载、实际投入填写与保存仍只依赖本地数据库。
