## ADDED Requirements

### Requirement: 海外换汇需求必须从权威建议金额派生
服务端 SHALL 从现有权威月度建议金额派生海外直投的人民币资金需求，并 MUST NOT 在前端复制或重新实现分配算法。

#### Scenario: 前端请求年度换汇预测
- **WHEN** API 收到有效计划和预测窗口
- **THEN** 服务端 SHALL 使用与月度建议相同的金额精度与舍入规则生成逐月基础金额

### Requirement: 换汇分类必须依据标的市场档案
服务端 MUST 根据 `execution_mode`、`settlement_currency` 与 `fx_quota_relevant` 分类每个建议金额，并 MUST 排除 QDII、港股通和本地标的。

#### Scenario: 名为“纳斯达克100指数（美股）”但执行方式为 QDII
- **WHEN** 该标的市场档案为 `qdii` 且结算币种为 `CNY`
- **THEN** 服务端 MUST 不将其计入 USD 需求

#### Scenario: 名为港股但执行方式未知
- **WHEN** 标的没有有效市场档案
- **THEN** 服务端 MUST 不根据名称推断 HKD 需求
- **AND** SHALL 返回可诊断的未分类状态

### Requirement: 年度聚合必须保留可解释明细
服务端 SHALL 为每个月和币种返回汇总金额及参与汇总的标的明细，使总额可由同一响应中的明细复算。

#### Scenario: 校验币种月度汇总
- **WHEN** 响应包含某月 USD 人民币需求
- **THEN** 该值 MUST 等于该月所有符合条件 USD 标的明细之和

### Requirement: 未知月份不得污染后续累计值
当预测窗口内任一月份缺少权威建议数据时，服务端 MUST 标记该月份及依赖它的累计值为未知，而 MUST NOT 将未知月份当作零。

#### Scenario: 年中一个月份不可计算
- **WHEN** 7 月建议数据不可计算
- **THEN** 7 月需求 MUST 为未知
- **AND** 7 月起依赖完整序列的累计曲线值 MUST 为未知或明确标记为部分累计
