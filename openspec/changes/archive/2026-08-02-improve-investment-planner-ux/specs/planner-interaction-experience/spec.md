## ADDED Requirements

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
系统 SHALL 在计划配置和月度录入流程中突出唯一主要动作，并根据当前计划状态说明用户可执行的下一步。

#### Scenario: Plan configuration is incomplete
- **WHEN** 用户尚未配置至少一个启用标的或启用标的比例未达到 100%
- **THEN** 系统 SHALL 展示差额和修正入口，并在启用计划前阻止无效提交

#### Scenario: Plan configuration is ready
- **WHEN** 启用标的名称有效且比例合计为 100%
- **THEN** 系统 SHALL 明确标记配置已就绪，并允许用户保存和进入月度录入

### Requirement: Organize investment targets predictably
系统 SHALL 支持用户按投资标的比例从高到低排序，也 SHALL 支持通过拖拽手动调整顺序，并在保存时提交与视觉顺序一致的 `sortOrder`。系统 SHALL 保留可通过键盘操作的上移、下移控件作为拖拽的替代方式。

#### Scenario: Sort targets by allocation
- **WHEN** 用户在投资标的设置中选择按比例排序
- **THEN** 系统 SHALL 将启用标的按分配比例从高到低排列，并稳定保留比例相同项的原有先后顺序

#### Scenario: Drag a target to a new position
- **WHEN** 用户将一个投资标的拖到另一个标的的位置
- **THEN** 系统 SHALL 立即更新显示顺序和各项 `sortOrder`，且保存后的概览与后续月度计算使用该顺序

### Requirement: Complete configuration with clear navigation and appearance controls
系统 SHALL 使用稳定的计划 ID 路径在概览、设置、月度记录、历史记录和统计 Tab 之间切换。设置保存成功后系统 SHALL 展示确认弹窗，并在用户确认后返回该计划概览。应用 SHALL 提供至少三套可切换皮肤并记住用户选择。

#### Scenario: Confirm a successful settings save
- **WHEN** 用户成功保存计划设置
- **THEN** 系统 SHALL 展示设置已保存的确认弹窗；用户点击确认后 SHALL 返回该计划概览

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
