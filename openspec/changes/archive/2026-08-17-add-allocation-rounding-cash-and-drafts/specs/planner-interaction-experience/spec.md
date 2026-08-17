## MODIFIED Requirements

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

## ADDED Requirements

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
