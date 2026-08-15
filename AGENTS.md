# AGENTS.md

版本：2.0
编码：UTF-8
适用范围：本文件适用于 `D:\agent` 工作区所有代码和文档。


## 零、操作前规则读取

- 每次收到新的用户任务、补充要求或任务方向变化后，在执行任何文件检索、读取、命令、编辑、构建、测试、提交或推送之前，必须重新读取仓库根目录的 `AGENTS.md`，并读取本次目标路径层级中所有适用的 `AGENTS.md`。
- 不得仅依赖历史对话、上下文摘要或之前读取过的规则；每一轮任务操作都必须以工作区当前版本的 `AGENTS.md` 为准。
- 任务涉及多个目录时，必须在操作对应目录前先读取该目录适用的 `AGENTS.md`；规则冲突时遵循作用范围更具体且不违反更高优先级指令的规则。


## 一、AI编程行为准则（核心）

### 1. 先思考后编码
- 主动说出对任务的假设，遇到不明确处必须提问，不可自行猜测。
- 如果发现更简单直接的路径，应主动指出。

### 2. 简单优先
- 严格遵循YAGNI原则，不实现未被要求的功能。
- 能用50行代码实现的功能，绝不写200行。
- 不为一次性代码创建复杂的抽象层。

### 3. 精准修改
- 改动需像外科手术一样精准，只修改与当前任务直接相关的代码。
- 不要顺手重写格式、注释或"优化"没坏的代码。

### 4. 目标驱动执行
- 将模糊命令转化为可验证的目标。例如：不说"修复bug"，而说"先编写复现bug的测试，再修改代码直到测试通过"。
- 核心精神：不要告诉Agent怎么做，而是告诉它成功的标准是什么。


## 二、代码文档与命名规范

### 语言约定
- 业务源码中的文档注释和解释性行内注释必须使用简体中文。
- 协议名、库名、标识符和代码字面量可以保留英文。

### 后端（Go）
- 包、类型、函数、方法、常量、变量和局部业务状态都必须使用中文 GoDoc 或解释性注释。
- HTTP 处理函数要说明请求方法、路径、鉴权、请求参数与响应语义。
- 使用 `Get-ChildItem -Directory | Where-Object { Test-Path "$($_.FullName)\doc.go" } | ForEach-Object { go doc ".\$($_.Name)" }` 生成或检查文档。

### 前端（TypeScript/React）
- 类型、组件、钩子、服务函数、共享状态、模块级常量、页面内部函数、回调和局部变量都必须使用中文 JSDoc/TypeDoc 注释。
- 页面外层注释不能代替页面内部声明的注释。
- TypeDoc 配置位于 `frontend/typedoc.json`。

### 数据库（SQLite）
- 每张表、迁移新增列和索引都要在迁移 SQL 旁用中文说明用途、关联关系和访问规则。

### 命名规则
- 优先使用 `recipientID`、`attachmentIDs`、`visitorLogRetentionDays` 等能表达领域含义的名称。
- 禁止新增 `data`、`info`、`item`、`handle` 等含义宽泛的名称。
- 注释必须说明变量保存的业务内容、函数执行的行为或状态变化，不得用"保存数据""处理信息"之类无实际信息的注释充数。

### API文档
- HTTP 契约维护在 `docs/api/openapi.yaml`。


## 三、项目定位与技术栈

### 整体定位
这是一个采集数据管理平台的全栈工作区，两个主要应用可以独立开发并通过 HTTP 联调。

### 后端（backend/）
- **语言**：Go 1.26
- **框架**：Gin
- **数据库**：SQLite
- **职责**：认证、会话、部门/角色/用户权限、业务数据持久化、文件存储
- **默认端口**：8080

### 前端（frontend/）
- **框架**：Next.js App Router + React + TypeScript
- **职责**：登录、工作台、用户、部门、角色、菜单权限、文章、文件管理
- **默认端口**：3000

### 重要约束
- 这不是 monorepo 工具链工程：根目录没有统一的构建脚本，命令必须在对应目录执行。
- 根目录的 `package-lock.json` 是遗留文件，不表示应在根目录安装前端依赖。


## 四、核心业务模型

### 会话与认证
- 登录会话由后端生成随机会话 ID，写入 HttpOnly Cookie，并持久化到 SQLite `sessions` 表。
- 前端不保存访问令牌。

### 权限体系
- **菜单权限**：决定用户能进入哪些页面。
- **动作权限**：使用 `resource.action` 编码控制查询、查看、创建、编辑、删除、授权等操作。
- **前端按钮隐藏只是体验优化，后端中间件才是安全边界**。
- 用户有效菜单 = 启用的直属部门菜单 ∪ 启用的角色菜单 ∪ 个人附加菜单，并递归补齐父级菜单。
- 用户有效动作 = 角色默认动作 ∪ 个人附加动作。
- `super-admin` 与 `system-admin` 固定拥有全部动作和菜单。
- 只有 `super-admin` 可创建、分配或修改超级管理员。
- 安全判断使用不可变 `roleCode`，不得依赖可编辑的角色显示名称。

### 资源权限（文章与文件）
- 都有 `ownerId` 和 `isPrivate` 字段。
- 公开资源：对有相应菜单/动作权限的用户可见。
- 私密资源：仅所有者或管理员可见。
- 修改和删除仍要求所有者或管理员。
- 文件删除默认是软删除（通过 `deletedAt` 进入回收站）。
- 永久删除会同时删除 SQLite 记录和物理文件，是高风险操作。


## 五、主要运行链路

### 后端启动顺序（固定）
加载配置 → 打开 SQLite → 迁移与幂等种子 → 补录上传目录中缺少数据库记录的文件 → 初始化会话与密码验证码服务 → 注册 Gin 路由。
- 迁移或种子失败时服务不会启动，禁止通过删除数据库来"修复"。

### 前端路由
- App Router 根路由为 `/`，登录后由 `useAdminWorkspace` 和 `activePage` 在客户端切换各管理页面。
- 功能页不是独立 URL，刷新位置恢复依赖 `sessionStorage`。
- 侧栏展开状态根据当前页面和异步菜单树推导。


## 六、跨目录约定

| 约定项 | 说明 |
|--------|------|
| 后端健康检查 | `GET /health` |
| 业务API | 统一位于 `/api` |
| 前端代理 | 通过 `NEXT_PUBLIC_API_BASE_URL` 覆盖后端地址 |
| 登录状态 | 由后端 HttpOnly Cookie 维护，前端请求必须携带凭证 |
| 开发环境CORS | 默认接受并回显任意 Origin |
| 生产环境CORS | 必须通过 `CORS_ALLOWED_ORIGINS` 使用明确的 Origin 白名单 |
| JSON字段格式 | camelCase |
| 用户可见文案 | 默认使用简体中文 |
| 依赖管理 | Go 命令在 `backend/` 执行，npm 命令在 `frontend/` 执行 |


## 七、内部聊天与客服聊天边界

| 模块 | 文件路径 | API前缀 |
|------|----------|---------|
| 内部聊天 | `frontend/app/chat/page.tsx` | `/api/internal-chat/*` |
| 客服聊天 | `frontend/src/features/chat/CustomerChatPage.tsx` | 独立socket/接口 |

### 关键约束
- 两套实现不得混用业务状态或鉴权规则。
- 内部聊天附件必须通过会话参与者鉴权的接口下载或预览，禁止使用公开静态地址暴露物理文件。
- 内部聊天实时事件使用认证后的 `/api/internal-chat/socket`。
- 新增消息必须同步驱动首页未读角标、聊天页视觉/声音提示和在线状态。
- 用户查看历史消息时，不得因 WebSocket 消息或 DOM 更新强制滚到底部；只有用户接近底部或主动查看最新消息时才自动跟随。


## 八、折叠与侧栏动画规范

- 新增或修改折叠面板、侧栏及其展开/收起操作时，必须提供清晰、平滑且不过度拖沓的过渡动画。
- 避免宽度、位移或内容状态瞬间跳变。
- 必须兼顾桌面端与移动端，不得引入横向溢出、内容遮挡或不可点击状态。
- 必须通过 `prefers-reduced-motion` 为减少动态效果的用户关闭或显著弱化动画。


## 九、访问分析与管理页面视觉约定

### 访问分析页面
- 文件位置：`frontend/src/admin-pages/visitor-analytics/VisitorAnalyticsPage.tsx`
- 菜单入口：`visitor-analytics`
- 展示顺序：`created_at DESC, id DESC`
- 分页：默认10条，固定支持 `10`、`20`、`30`、`50`、`100`
- 筛选控件：统一高度、垂直居中、保持明显间距，窄屏自动单列且不得横向溢出
- 统计数字：使用平滑数字动画
- 统计卡/图表卡/明细卡：可使用统一的3D hover效果
- 访问者列：图标与文字必须居中
- 隐私说明：不能被误解为"当前没有数据"

### 全局视觉
- 内部聊天 `/chat` 与客服聊天页面均应尽量使用可用内容宽度，避免两侧无意义的大块留白。
- 文件管理必须保留刷新数据和"设为登录背景"操作。
- Ant Design 弃用属性应按当前版本 API 更新，例如使用 `mask.closable` 替代 `maskClosable`。


## 十、改动联动矩阵

| 改动类型 | 必查位置 |
|----------|----------|
| API 路径或 HTTP 方法 | `backend/routes/`、对应 `handlers/`、`frontend/app/hooks/useAdminWorkspace.ts` 或 `frontend/app/lib/`、`README.md` |
| 请求/响应字段 | `backend/models/models.go`、repository 扫描/写入、handler、`frontend/app/types/admin.ts`、所有调用方 |
| 菜单或页面 | 后端菜单种子与 `RequireMenu`、前端 `PageKey/pageKeys/pageTitles`、`MainLayout` 图标/页面映射、`app/page.tsx` |
| 动作权限 | `backend/permissions/actions.go`、路由 `RequireAction`、repository 有效权限合并、`frontend/app/lib/actionPermissions.ts`、按钮显隐 |
| 角色/部门规则 | repository 迁移与种子、用户关联名称同步、保护性测试、前端角色/部门选择器 |
| SQLite 表或迁移 | `repository/sqlite_store.go`、扫描列顺序、幂等迁移测试；不得手改正式数据库 |
| 文件能力 | 后端路由/handler/repository、`frontend/app/lib/fileApi.ts`、`FilesPage.tsx`、上传限制与回收站语义 |
| 文章导出 | 后端批量 CSV/PDF 导出与前端单篇导出是两条独立路径；同时检查 `articleExport.ts` 和 `articleMarkdown.ts` |
| 主题或全局视觉 | `frontend/app/theme/themes.ts`、`globals.css`、Ant Design token、桌面/移动端验收 |


## 十一、开发与验证命令

### 后端
```powershell
cd backend
go test ./...
go vet ./...
go run .
