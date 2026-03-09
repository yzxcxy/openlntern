# openIntern Frontend Refactor Plan

## 1. 当前前端风格问题分析

### 1.1 视觉层问题
- 视觉基调偏“工具后台默认风格”，年轻化表达不足，整体以 `gray/blue` 的基础色为主，缺少品牌记忆点与情绪层次。
- 色彩来源分散：
  - `app/globals.css` 仅定义了基础前景/背景变量。
  - `app/a2ui-theme.css` 独立维护了另一套完整色板与 `--primary`。
  - 页面中大量使用硬编码 Tailwind 色值（如 `text-gray-500`、`bg-blue-600` 等）。
- 暗黑模式不可用或不完整：虽然有 `prefers-color-scheme`，但页面大量硬编码浅色类名，实际无法形成稳定暗黑体验。

### 1.2 组件一致性问题
- UI 方案混用：
  - 聊天/知识库页使用 `@douyinfe/semi-ui-19` 组件。
  - 登录、用户、技能、A2UI 等页面大量使用原生 `button/input/select/textarea` + Tailwind。
- 基础组件重复实现：存在自定义 `Modal`、`ConfirmDialog`，同时又使用 Semi 的 `Modal`，导致弹窗体验不一致。
- 交互反馈不统一：按钮 hover/focus/disabled 风格和强度在不同页面差异明显。

### 1.3 设计系统缺失问题
- 无统一 Design Token（颜色、圆角、阴影、间距、动效），样式语义无法复用。
- 无组件分层（Primitive/Base/Business），页面直接堆叠样式，长期维护成本高。
- 无 Storybook/组件文档与视觉回归机制，重构风险无法量化控制。

### 1.4 可维护性与工程风险
- 页面样式分散，修改品牌色或控件样式需要跨多个页面逐一改动。
- 多字体来源并行（本地字体 + Google Fonts），排版策略不统一。
- 视觉升级无法通过“批量替换 Token + 组件升级”完成，只能靠逐页手工调整。

## 2. 重构设计原则

### 2.1 设计语言原则
- Token First：先统一设计 Token，再驱动组件和页面改造。
- Component First：页面不得直接定义“品牌级样式”，优先通过统一组件输出。
- Progressive Refactor：按模块渐进替换，避免一次性大改导致功能回归。
- Accessibility by Default：颜色对比、焦点态、键盘可达性作为默认约束。

### 2.2 年轻化策略
- 视觉关键词：轻盈、清晰、科技感、积极。
- 通过“明亮主色 + 有节奏的中性色 + 克制高亮强调色”建立活力感。
- 减少大面积纯灰背景，增加低对比层次面与轻量渐变区域。
- 图标和卡片层级更鲜明，提升信息识别速度。

### 2.3 视觉基调
- 主基调：清爽浅色 + 高可读深文本。
- 强调方式：关键行为（CTA）使用品牌主色，次级行为保持中性。
- 节奏策略：统一 4px 网格、固定层级间距、明确标题/正文/注释字体阶梯。

## 3. 配色体系优化方案

### 3.1 目标
- 建立统一“语义色系统”，避免页面直接依赖具体色值。
- 支持未来暗黑模式扩展。

### 3.2 建议色板（Light）
- 主色 Primary：`#2563FF`（品牌主动作、主按钮、链接强调）
- 辅助色 Secondary：`#00BFA5`（次强调、标签、成功类弱化场景）
- 强调色 Accent：`#FF7A45`（运营活动、重点提示，低频使用）
- 中性色 Neutral：
  - `Neutral-900 #0F172A`（主文本）
  - `Neutral-700 #334155`（次文本）
  - `Neutral-500 #64748B`（辅助文本）
  - `Neutral-200 #E2E8F0`（边框）
  - `Neutral-100 #F1F5F9`（弱背景）
  - `Neutral-50 #F8FAFC`（页面底色）
- 功能色：
  - Success `#16A34A`
  - Warning `#F59E0B`
  - Error `#DC2626`
  - Info `#0EA5E9`

### 3.3 语义 Token（示例）
- `--color-bg-page` / `--color-bg-surface` / `--color-bg-elevated`
- `--color-text-primary` / `--color-text-secondary` / `--color-text-muted`
- `--color-border-default` / `--color-border-strong`
- `--color-action-primary` / `--color-action-primary-hover`
- `--color-state-success|warning|error|info`

### 3.4 暗黑模式（可选）
- 暂不在第一阶段全量切换，仅完成 Token 预留：
  - 先提供 `data-theme="dark"` 的 token map。
  - 首批覆盖聊天页与侧边栏两大高频区域。
  - 通过视觉回归测试控制色差风险。

## 4. 组件体系统一方案

### 4.1 核心组件清单（优先级从高到低）
- 布局：`AppShell`、`Sidebar`、`TopBar`、`PageContainer`
- 操作：`Button`、`IconButton`、`DropdownMenu`
- 数据展示：`Card`、`Tag`、`Badge`、`EmptyState`、`Toast`
- 表单：`Input`、`Textarea`、`Select`、`Checkbox`、`FormField`
- 反馈与浮层：`Modal`、`ConfirmDialog`、`Loading`、`Skeleton`
- 导航：`NavItem`、`Tabs`、`Breadcrumb`

### 4.2 统一规范
- 圆角：`6 / 10 / 14 / 18` 四档（输入框 10，卡片 14，弹窗 18）
- 阴影：
  - `shadow-sm`：列表项/轻卡片
  - `shadow-md`：弹层/浮层
  - `shadow-lg`：关键浮窗（严格限用）
- 间距：采用 4px 网格；组件内部优先 `8/12/16/20/24`
- 边框：默认 1px，颜色只使用语义 token
- 字体层级：
  - 标题 `20/24`，子标题 `16/20`，正文 `14/20`，注释 `12/16`
- 交互反馈：
  - Hover：亮度微调 + 背景过渡（150ms）
  - Active：缩放 `0.98`（仅按钮）
  - Focus：统一 `2px` 品牌色 focus ring
  - Disabled：透明度 + 禁止指针

### 4.3 兼容 Semi UI 的过渡策略
- 不立即移除 Semi；先封装 `ui-adapter`：
  - `UiButton`（内部可切 Semi 或本地 Button）
  - `UiModal`、`UiInput` 同理
- 新页面只允许使用统一组件层，不直接使用 Semi 原生组件。

## 5. 页面层级与布局优化建议

### 5.1 全局布局
- 建立统一页面容器：
  - 页面外层使用 `PageContainer`（固定最大宽度 + 自适应留白）。
  - 内容区分为 `Header / Content / FooterAction` 三段。
- 侧边栏与内容区对比增强：
  - 侧边栏背景与主内容背景做轻度区分，减少视觉平铺。
  - 当前导航项使用品牌弱色底 + 左侧强调条。

### 5.2 信息层级
- 每页明确“主任务 CTA”最多 1-2 个，其他操作降级为次要按钮。
- 卡片内结构统一为“标题 > 元信息 > 主内容 > 操作区”。
- 异常提示（错误/成功）统一采用 Alert 组件，不再手写散落样式。

### 5.3 排版与留白
- 移除过密区域（例如列表和卡片之间固定 `16/20` 间距）。
- 标题和正文间建立固定节奏，避免每页自定义 margin。
- 大段内容页（技能详情）使用阅读宽度限制（例如 760-860px）提升可读性。

## 6. 动效与微交互设计建议

### 6.1 动效原则
- 目标是“提高反馈感”，不是“炫技”。
- 单次动画时长控制在 `120-220ms`。
- 统一缓动曲线：`cubic-bezier(0.2, 0, 0, 1)`。

### 6.2 建议落地场景
- 导航切换：选中态平滑过渡 + 左侧高亮条滑入。
- 卡片悬停：轻微上浮（`translateY(-1px)`）+ 阴影增强。
- Modal：淡入 + 轻微上移。
- 表单校验：错误提示淡入，输入框边框过渡。
- 列表分页/加载：使用 Skeleton 替代纯文本“加载中”。

### 6.3 性能与可访问性约束
- 遵循 `prefers-reduced-motion`，在该模式下降级动画。
- 禁止连续复杂动画叠加，避免输入和滚动场景掉帧。

## 7. 技术落地方案

### 7.1 技术选型建议
- 保持 `Next.js + Tailwind` 主栈不变，降低重构成本。
- 增加设计系统基础工具：
  - `class-variance-authority`：管理组件变体
  - `tailwind-merge`：消除 class 冲突
  - `clsx`：条件样式可读化
- 建议引入 Storybook（仅前端本仓）：
  - 输出 Button/Card/Form/Modal 视觉规范与交互状态。

### 7.2 Token 化改造
- 新增 `app/styles/tokens.css`：沉淀颜色、圆角、阴影、动效、间距 token。
- 在 `tailwind.config.ts` 中扩展 `theme.colors/radius/shadow` 映射到 CSS Variables。
- 页面禁止直接写品牌具体色值（如 `blue-600`），改用语义 class/token。

### 7.3 代码组织建议
- 新增目录：
  - `app/components/ui/*`（基础组件）
  - `app/components/layout/*`（布局组件）
  - `app/styles/tokens.css`（设计 token）
  - `app/styles/motion.css`（统一动效）
- ESLint 规则新增：限制直接使用特定硬编码色 class（可先 warning，后 error）。

### 7.4 质量保障
- Storybook 快照（Chromatic 或本地截图基线）
- 关键流程 E2E（登录、聊天、知识库、技能管理）回归
- 可视一致性检查清单：颜色、间距、圆角、交互态、可读性

## 8. AI 执行路线图（低风险流水线）

### Phase 0：基线采集与规则冻结
- 触发条件：首次执行重构任务。
- AI 输入：当前代码库、页面清单、已有样式文件。
- AI 动作：
  - 自动扫描硬编码颜色、原生控件使用点、Semi 组件使用点。
  - 生成基线报告（页面、组件、样式热点）。
  - 输出 token 草案与组件 API 草案文档。
- AI 输出：`audit-report`、`tokens-v1-draft`、`component-api-draft`。
- 通过门禁：报告文件生成且覆盖登录/工作台/聊天/知识库/技能页。
- 失败回滚：仅文档产物，无代码回滚需求。

### Phase 1：Token 与基础组件初始化
- 触发条件：Phase 0 通过。
- AI 输入：`tokens-v1-draft`、基础组件清单。
- AI 动作：
  - 新建 `tokens.css` 与 Tailwind token 映射。
  - 实现 `Button/Input/Card/Modal/Alert` 基础组件。
  - 在 Storybook 中输出组件状态（default/hover/focus/disabled）。
- AI 输出：可编译的 token 与基础组件库。
- 通过门禁：
  - 基础组件示例可渲染。
  - 登录/注册页可无损迁移到新组件。
- 失败回滚：按 PR 颗粒回退到“仅保留 token 文件”状态。

### Phase 2：工作台骨架迁移
- 触发条件：Phase 1 通过。
- AI 输入：`AppShell/Sidebar/PageContainer` 目标规范。
- AI 动作：
  - 重构 `(workspace)/layout`，替换散装按钮/输入/弹窗样式。
  - 统一导航态、侧边栏、内容容器层级与间距节奏。
  - 保留业务逻辑与接口调用，不改请求协议。
- AI 输出：统一骨架布局组件及工作台接入结果。
- 通过门禁：
  - 侧边栏与主内容区仅消费语义 token。
  - 快捷入口、历史会话、用户信息区交互保持原行为。
- 失败回滚：回退布局组件改动，不影响业务数据流。

### Phase 3：业务页面批量迁移
- 触发条件：Phase 2 通过。
- AI 输入：技能/A2UI/用户页面与组件映射表。
- AI 动作：
  - 逐页替换原生控件为统一组件。
  - 新增 `ui-adapter`，隔离 Semi 与自研组件差异。
  - 删除重复样式块，统一 alert、表单、卡片样式语义。
- AI 输出：页面级统一组件接入版本。
- 通过门禁：
  - 页面不再新增硬编码品牌色 class。
  - 页面关键链路（列表、编辑、删除、上传）可用。
- 失败回滚：按页面维度回滚，保证任一页面可独立恢复。

### Phase 4：复杂页面精修（聊天/知识库）
- 触发条件：Phase 3 通过。
- AI 输入：聊天页、知识库页现有 Semi 组件与主题覆盖点。
- AI 动作：
  - 对齐 Semi 主题与 token 映射，减少局部冲突样式。
  - 补齐动效与微交互（导航态、弹窗、加载态、错误提示）。
  - 增加 `prefers-reduced-motion` 兼容策略。
- AI 输出：复杂页视觉统一版本。
- 通过门禁：
  - 聊天流、知识库树、上传/删除/创建流程行为不退化。
  - 视觉回归快照无关键差异。
- 失败回滚：优先回退主题覆盖层，保留业务功能代码。

### Phase 5：收口、约束固化与持续执行
- 触发条件：Phase 4 通过。
- AI 输入：全量页面与组件统计数据。
- AI 动作：
  - 生成重构结果报告与遗留问题清单。
  - 固化 ESLint 规则（限制硬编码色/散装控件）。
  - 建立后续新增页面的自动检查模板。
- AI 输出：`refactor-report`、`residual-backlog`、`style-guard-rules`。
- 通过门禁：
  - 新增 PR 可自动检测样式违规。
  - 组件复用率与硬编码下降率可度量。
- 失败回滚：规则先以 warning 运行，再逐步提升到 error。

---

## 附：AI 执行约束
- 新增 UI 代码默认接入统一组件层，不允许继续新增页面内通用样式实现。
- 风格变更顺序固定为：Token -> 组件 -> 页面，禁止反向改动。
- 每次执行必须是可回滚的最小提交单元，并附带自动验证结果。
