## ADDED Requirements

### Requirement: 参与换汇预测的标的必须按执行方式确定
系统 MUST 只把 `direct_us` 和 `direct_hk` 且 `fx_quota_relevant=true` 的计划标的纳入换汇预测，并 MUST 排除 QDII、港股通和其他人民币结算标的。

#### Scenario: 计划只有 QDII 美股指数基金
- **WHEN** 计划没有美股直投或港股直投标的
- **THEN** 系统 MUST 不生成 USD 或 HKD 换汇需求

#### Scenario: 计划包含港股通标的
- **WHEN** 标的通过港股通以人民币结算
- **THEN** 该标的 MUST 不计入个人 HKD 换汇需求

### Requirement: 预测币种必须由计划实际标的动态决定
系统 SHALL 仅返回当前计划真实需要的币种，不得返回金额为零的无关币种占位。

#### Scenario: 只有美股直投
- **WHEN** 计划包含美股直投但不包含港股直投
- **THEN** 响应 SHALL 包含 USD
- **AND** MUST 完全省略 HKD 的卡片、曲线和汇总字段

#### Scenario: 只有港股直投
- **WHEN** 计划包含港股直投但不包含美股直投
- **THEN** 响应 SHALL 包含 HKD
- **AND** MUST 完全省略 USD 的卡片、曲线和汇总字段

#### Scenario: 两种直投都不存在
- **WHEN** 计划没有任何需个人换汇的海外直投标的
- **THEN** 系统 SHALL 返回空的币种集合
- **AND** 月度记录页 MUST 不显示年度换汇预测入口

### Requirement: 年度需求必须从服务端月度建议派生
系统 SHALL 使用与月度记录一致的服务端计算规则生成预测窗口内逐月人民币需求，并 MUST 按币种汇总计划内符合条件的实际标的。

#### Scenario: 多个美股标的分布在同一计划
- **WHEN** 多个 `direct_us` 标的在同一月份有建议投入
- **THEN** 该月 USD 人民币需求 SHALL 等于这些实际标的建议投入之和
- **AND** 响应 SHALL 保留参与汇总的标的标识以便解释

#### Scenario: 切换预测窗口
- **WHEN** 用户在“下一自然年”和“滚动 12 个月”之间切换
- **THEN** 系统 SHALL 使用对应月份重新计算需求序列

### Requirement: 外币金额必须使用可追溯规划汇率
系统 SHALL 使用对应币种的规划汇率将人民币需求换算为外币估算值，并 MUST 返回汇率来源、观测日期和假设类型。

#### Scenario: 使用最新人民币中间价规划未来月份
- **WHEN** 未来月份尚无实际汇率且最新中间价有效
- **THEN** 系统 SHALL 使用该中间价作为静态规划假设
- **AND** 界面 MUST 标记结果为估算而非成交价

#### Scenario: 用户输入情景汇率
- **WHEN** 用户为币种输入有效的情景汇率
- **THEN** 系统 SHALL 使用情景值重新计算该币种结果
- **AND** MUST 标记来源为用户情景值

#### Scenario: 汇率不可用
- **WHEN** 某个所需币种没有有效汇率或情景覆盖
- **THEN** 该币种外币金额和 USD 等值额度占用 MUST 显示 `--`
- **AND** 其人民币逐月需求 SHALL 继续显示

### Requirement: 年度便利化额度只能作为合规提示下的参考阈值
系统 SHALL 将官方年度便利化额度作为带来源和生效日期的参考阈值，并 MUST 说明该额度不是绝对购汇上限，也不代表境外证券投资许可。

#### Scenario: 展示额度占用曲线
- **WHEN** 用户查看年度累计 USD 等值需求
- **THEN** 界面 SHALL 将 50,000 USD 标记为“年度便利化额度参考”
- **AND** MUST 展示合规提示与官方来源日期

#### Scenario: 预计需求超过参考阈值
- **WHEN** 累计 USD 等值估算超过当前参考阈值
- **THEN** 系统 SHALL 显示超出提示
- **AND** MUST NOT 表述为系统允许、禁止或可代办购汇

### Requirement: 缺失数据不得被伪装成零需求
系统 MUST 区分真实的零需求与无法计算的未知需求。

#### Scenario: 某月符合条件的标的建议投入为零
- **WHEN** 服务端明确计算该币种当月需求为零
- **THEN** 系统 SHALL 显示真实的 `0`

#### Scenario: 月度建议记录缺失或计算失败
- **WHEN** 预测月份缺少必要计划数据且无法计算
- **THEN** 系统 MUST 显示 `--`
- **AND** MUST NOT 将该月计为零后继续给出虚假的完整年度累计值
