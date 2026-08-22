# AGENTS.md

版本：3.2
编码：UTF-8
适用范围：本文件适用于 `D:\agent` 工作区全部代码、文档和运行配置；子目录 `AGENTS.md` 在其作用域内补充本文件。

## 零、操作前规则读取

- 每次收到新的用户任务、补充要求或任务方向变化后，在执行任何文件检索、读取、命令、编辑、构建、测试、提交或推送之前，必须重新读取本文件，并读取本次目标路径层级中所有适用的 `AGENTS.md`。
- 不得仅依赖历史对话、上下文摘要或之前读取过的规则；每一轮任务操作都必须以工作区当前版本的规则为准。
- 任务涉及多个目录时，必须在操作对应目录前先读取该目录的 `AGENTS.md`。规则冲突时遵循作用范围更具体且不违反更高优先级指令的规则。

## 一、工作原则

### 1. 先确认目标

- 主动说明关键假设；只有会实质改变结果且无法从仓库确认的问题才向用户提问。
- 把需求转换为可验证目标，完成实现后继续执行相称的测试和验收。

### 2. 简单与精准

- 遵循 YAGNI，优先使用项目已有框架、目录和辅助函数，不为单次改动增加多余抽象。
- 只修改当前任务直接涉及的文件，不顺手格式化、重构或回退无关内容。
- 工作区已有改动默认属于用户；与任务无关时保留并忽略，与任务重叠时在其基础上继续修改。

### 3. 错误恢复

- 命令、工具、网络、构建或测试报错后，立即检查任务是否完成，修复原因并重试可安全重试的步骤。
- 对提交、推送、创建数据和删除等可能产生副作用的操作，重试前先核对当前状态，避免重复执行。

## 二、当前项目状态

这是一个采集数据管理平台。仓库目前有三个可独立运行的应用：

| 目录 | 当前状态 | 技术栈与职责 | 默认地址 |
| --- | --- | --- | --- |
| `backend/` | 可运行 | Go 1.26、Gin、SQLite、Redis；认证、权限、内容、文件、聊天、监控、SSH 与宿主机代理 | `http://localhost:8080` |
| `frontend/` | 可运行 | Next.js 16 App Router、React、strict TypeScript、Ant Design、Tailwind CSS 4 | `http://localhost:3000` |
| `portal/` | 可运行 | Next.js 16 App Router、React、strict TypeScript、Tailwind CSS 4、next-intl；公开内容门户 | `http://localhost:3001` |

- 根目录没有统一构建脚本，也不是 monorepo 工具链工程；Go 和 npm 命令必须在对应应用目录执行。
- 根 `package-lock.json` 是被 `.gitignore` 忽略的遗留文件，不表示应在根目录安装依赖。
- `docker-compose.yml` 编排 Redis、后端、管理前端和 C 端门户；三个应用仍分别维护依赖和构建命令。

## 三、目录与运行链路

### 后端

- 启动顺序固定为：加载配置 -> 打开 SQLite -> 增量迁移与幂等种子 -> 补录上传目录 -> 初始化会话与密码验证码服务 -> 注册 Gin 路由。
- 迁移或种子失败时禁止通过删除数据库修复。`backend/store/` 是遗留内存实现，生产装配使用 `backend/repository/` 的 SQLite 实现。
- HTTP 契约维护在 `docs/api/openapi.yaml`，数据库表说明维护在 `docs/database/schema.md`。

### 管理前端

- `frontend/app/` 只承载 App Router 入口和全局样式；管理页面位于 `frontend/src/admin-pages/`，跨页面业务状态位于 `frontend/src/features/`。
- 根管理页 `/` 登录后由 `frontend/src/features/workspace/useAdminWorkspace.ts` 和 `activePage` 切换页面；`/chat` 与 `/socket/chat/[conversationId]` 是独立真实路由。
- 页面恢复使用 `sessionStorage`，主题使用独立的 `localStorage` 键；后端 HttpOnly Cookie 是后台认证的唯一凭据。

### C 端门户

- `portal/` 使用真实的 `/{locale}/...` App Router 路由，支持 `zh-CN`、`en-US`、`ja-JP`、四种主题、SEO、公开内容检索、图片预览和登录后 18R 开关。
- C 端业务数据和图片互动只能来自 `/api/public/*`；不得直连 SQLite、扫描上传目录或调用后台写接口。登录复用后端 HttpOnly Cookie 会话，用于身份恢复、18R 可见性、图片点赞和评论，不在浏览器持久化会话 ID。
- C 端图片页不显示分页按钮，但后端分页契约仍作为内部批次协议；接近底部时自动预加载下一批，稳定瀑布流列不得因追加数据重排已加载卡片。
- 图片列表默认每批 24 条、最多 50 条；缩略图接口返回后端生成的最大 `480x480` JPEG，中图最大宽度 `1280`，原图只在预览浮层按需读取。图片加载队列在移动端最多并发 2 张、宽视口最多并发 4 张，只取消仍在请求中的预加载项，不卸载已加载并挂载的卡片。
- 预览浮层支持复制当前原图 URL、下载原图、点赞切换、标签展示和评论；互动读取可匿名，点赞与评论要求有效登录，评论为去首尾空白的纯文本且最多 500 个 Unicode 字符。

## 四、代码、注释与命名

- 业务源码的文档注释和解释性行内注释使用简体中文；协议名、库名、标识符和代码字面量可保留英文。
- Go 包、类型、函数、方法、常量、变量和局部业务状态使用中文 GoDoc 或解释性注释；Gin handler 还要说明方法、路径、鉴权、请求与响应语义。
- TypeScript 类型、组件、钩子、服务函数、共享状态、模块级常量、页面内部函数、回调和局部业务变量使用中文 JSDoc/TypeDoc 或解释性注释。
- 优先使用 `recipientID`、`attachmentIDs`、`visitorLogRetentionDays` 等领域名称，避免新增 `data`、`info`、`item`、`handle` 等宽泛命名。
- 注释必须表达业务用途、状态变化或安全边界，不能只是复述语法。
- SQLite 新表、迁移列和索引必须在迁移 SQL 附近使用中文说明，并同步模型、扫描顺序、handler、OpenAPI 和测试。

## 五、认证与权限边界

- 登录会话由后端生成随机 ID，写入 HttpOnly Cookie 并持久化到 `sessions`；前端不得存储访问令牌、会话 ID 或密码。
- 有效菜单 = 启用的直属部门菜单 + 启用的角色菜单 + 个人附加菜单，并递归补齐父级菜单。
- 有效动作 = 角色默认动作 + 个人附加动作；`super-admin` 与 `system-admin` 固定拥有全部菜单和动作。
- 前端按钮隐藏仅用于体验，后端 `RequireAuth`、`RequireMenu`、`RequireAction` 和所有权校验才是安全边界。
- 安全判断只能使用不可变 `roleCode`，不得依赖角色显示名称或用户名。`MH` 只用于用户表首次为空时的初始化，不得在运行逻辑中获得特殊待遇。
- 超级管理员权限彼此相同，可以互相修改资料和登录权限；系统管理员及其他角色不得创建、分配、修改、停用或删除超级管理员。

## 六、核心业务边界

### 文章与文件

- 文章和文件均有所有者与私密状态；公开资源按菜单和动作权限读取，私密资源仅所有者或管理员读取，写入仍需所有权或管理员身份。
- 文件默认软删除到回收站。永久删除同时移除 SQLite 记录和物理文件，只有用户明确操作或授权后才能执行。
- C 端公开文章必须满足 `is_private=0 AND status='已发布'`；公开文件必须满足 `is_private=0 AND deleted_at IS NULL`。匿名访问额外排除 `is_18r=1`，只有有效登录会话且 `portal-r18=1` 时才包含 18R 内容。
- B 端文件管理支持一次选择多个文件；前端过滤同一批次的重复选择，后端以同一所有者的完整内容 SHA-256 对未删除文件做权威去重并返回 `409 DUPLICATE_FILE`。文件管理上传不设置应用层单文件大小上限，实际限制来自磁盘、代理和操作系统资源；聊天附件仍有独立限制。
- 文件标签在写入时去首尾空白、去掉前导 `#`、按不区分大小写去重，最多 12 个标签、每个最多 24 个 Unicode 字符。受保护的 `/api/files/:id/thumbnail` 可使用可再生成的 `.thumbnail-cache`，缓存不是业务文件，也不改变软删除语义。

### 两套聊天

| 模块 | 前端入口 | API |
| --- | --- | --- |
| 内部聊天 | `frontend/app/chat/` | `/api/internal-chat/*` |
| Socket 客服 | `frontend/src/features/chat/`、`frontend/app/socket/chat/` | `/api/socket/*` |

- 两套聊天不得混用业务状态、访客令牌或鉴权规则。
- 附件必须通过参与者或管理员鉴权的接口预览和下载，禁止暴露物理上传路径。
- 用户查看历史消息时，实时消息或 DOM 更新不得强制滚到底部；只有接近底部或主动查看最新消息时自动跟随。

### 服务器监控与 SSH

- `GET /api/server/metrics` 需要登录、`dashboard` 菜单和 `dashboard.view` 动作，返回后端运行环境视角的 CPU、内存、磁盘、网络、进程、温度和告警；容器内只能代表容器可见资源。
- `GET /api/server/connections` 使用相同权限按需返回活动连接明细；平台或进程权限不支持枚举时必须返回结构化不可用状态，不能把权限限制伪装成零连接。
- “业务资源”是工作台下独立的 `business-resources` 页面，展示用户、菜单和文章的总量、有效量、构成及可用率，不得重新塞回预览台底部长页面。
- `/api/server/terminal` 是任意有效登录用户可用的 SSH WebSocket，不得重新加入管理员限定。
- `/api/server/host-agent` 使用 `HOST_AGENT_TOKEN` 接受单个 Linux 宿主机代理主动注册；`/api/server/host-terminal` 只允许 `roleCode=super-admin`，不得向系统管理员或普通用户开放。
- 部署机直连命令和文件权限必须等于代理进程系统账号；当前受支持的生产方案是宿主机上以 `root` 运行独立 systemd 代理，因此仅向可信超级管理员开放。禁止通过特权容器、Docker Socket 或挂载宿主机根目录替代代理，也不得把容器 shell 标记为宿主机终端。
- SSH 凭据、私钥、文件内容和连接状态只保存在当前浏览器页面内存；弹窗最小化和页面切换保持连接，退出登录或关闭浏览器页面时销毁。
- 多终端、主机指纹确认、SFTP 步进目录、当前目录搜索、终端路径同步、文本编辑保存、图片/PDF 预览属于同一协议能力，修改一端时必须同步检查后端协议、前端会话状态和 OpenAPI。
- 部署机直连的生产部署账号是宿主机 `root`；更新 Compose 镜像时不要在前端终端中前台运行构建，使用动态名称的 `systemd-run --no-block` oneshot 单元，并通过 `systemctl status` 或 `journalctl -u <unit>` 查看结果。该模式不使用 SSH/22 端口；普通 SSH 模式仍由后端连接目标主机的 SSH 端口。

## 七、跨目录联动矩阵

| 改动类型 | 必查位置 |
| --- | --- |
| API 路径或 HTTP 方法 | `backend/routes/`、`backend/handlers/`、`frontend/src/services/`、所有调用方、`docs/api/openapi.yaml`、`README.md` |
| 请求或响应字段 | `backend/models/`、`backend/repository/`、handler、`frontend/src/types/`、调用方 |
| 菜单或管理页面 | 菜单种子、`RequireMenu`、`frontend/src/config/constants.ts`、`frontend/src/components/layout/MainLayout.tsx`、`frontend/app/page.tsx` |
| 动作权限 | `backend/permissions/actions.go`、路由中间件、有效权限合并、`frontend/src/utils/actionPermissions.ts`、按钮显隐 |
| 用户、角色、部门 | repository 事务与迁移、关联名称同步、保护性测试、管理前端选择器 |
| SQLite 迁移 | `backend/repository/sqlite_store.go`、相关 repository、扫描顺序、迁移幂等测试、`docs/database/schema.md` |
| 文件与聊天附件 | routes、handlers、repository、`frontend/src/services/fileApi.ts`、文件页、聊天调用方、回收站语义 |
| 公开门户能力 | `/api/public/*`、文章/文件公开与 18R 字段、OpenAPI、`docs/portal-requirements.md`、`portal/` 调用方 |
| 服务器与终端 | `backend/handlers/server.go`、`backend/handlers/host_agent.go`、`backend/cmd/host-agent/`、`backend/routes/server.go`、`frontend/src/services/serverApi.ts`、终端组件、OpenAPI |
| 主题、布局或交互 | `frontend/src/theme/`、`frontend/app/globals.css`、相关 CSS/组件、桌面和移动浏览器验收 |

## 八、数据与秘密保护

- `backend/data/`、业务 SQLite/WAL/SHM、`backend/uploads/` 及外部挂载的数据目录全部视为用户生产数据。
- 构建、测试、脚本和浏览器验收不得删除、覆盖、重置或清空业务数据，也不得通过真实业务 API 批量清理验收数据。
- 测试与联调必须同时使用独立临时 SQLite 和上传目录，统一放在 `.workspace-temp/<task>/`；不得用删除正式数据库解决迁移或启动问题。
- 文件删除默认采用软删除；永久物理删除、生产数据库修复和上传目录清理必须取得用户明确授权。
- 仓库根 `email.txt` 和环境变量中的 SMTP、Redis、Cookie 配置属于秘密，不得提交或输出真实值。Docker 将根 `email.txt` 只读挂载到 `/app/email.txt`。

## 九、界面与浏览器验收

- 沿用现有 Ant Design、Lucide、shadcn/ui、Tailwind 和原生 CSS 体系；常见工具操作优先使用已有图标库。
- 卡片圆角默认不超过 8px，不创建卡片嵌套卡片或装饰性光斑；文字、按钮和固定格式控件必须有稳定响应式尺寸。
- 同一工具区或分段控件内的按钮必须使用统一高度和视觉中心线，图标与文字作为一个 flex 整体居中；Header、Dialog、Segmented 等场景混用 Ant Design 与 Lucide 图标时统一包装器、行高和 SVG 显示方式，禁止用单个图标的 `top`、margin 或 `translateY` 掩盖结构性偏移。
- 修改侧栏、折叠、抽屉或内容显隐时提供平滑过渡，并在 `prefers-reduced-motion` 下关闭或弱化动画。
- 涉及视觉和交互时使用当前环境的官方 Browser 能力检查桌面与移动端、控制台错误、横向溢出、遮挡、焦点和真实交互结果。

## 十、启动与验证

### 后端

```powershell
cd backend
gofmt -w <修改的.go文件>
go test ./...
go vet ./...
go run .
```

### 管理前端

```powershell
cd frontend
.\node_modules\.bin\tsc.cmd --noEmit --incremental false
npm run build
npm run dev
```

- `frontend` 当前没有自动化测试脚本；`npm run lint` 仍调用 Next.js 16 已移除的 `next lint`，在脚本修复前不作为有效验收命令。
- 开发服务和生产构建共享 `.next`，不要并发执行；不要手工编辑或提交 `.next/`、`node_modules/`、`tsconfig.tsbuildinfo`、`next-env.d.ts` 等生成物。

### C 端门户

```powershell
cd portal
npm run typecheck
npm run lint
npm run docs
npm run build
npm run dev
```

- TypeDoc 输出到 `portal/docs/`，属于生成文档；业务规则和接口说明仍维护在根 `docs/`。
- 开发服务和生产构建共享 `.next`，不要并发执行；涉及布局、动画或交互时按本文件浏览器验收规则检查三语言和桌面/移动视口。

### Docker

```powershell
docker compose up --build
```

- Compose 默认启动 Redis、后端、管理前端和 C 端门户；持久化目录、端口和公开地址通过根 `.env` 或 `.env.example` 中的变量覆盖。

## 十一、Git 提交与推送

- 当前 `Pn` 取最近一次已创建的 `P<number>` 提交；同一批尚未提交的连续补充改动沿用同一个编号，前一任务提交后下一独立任务才递增。
- 提交摘要必须使用 `P<number> <type>: 中文具体摘要`，常用类型为 `fix`、`feat`、`docs`、`refactor`、`style`、`test`、`chore`、`perf`、`build`、`ci`。
- 提交正文或提交前报告必须说明关键改动、验证结果、隔离环境、残余风险和当前 `Pn`。
- 用户明确要求提交或推送前不得暂存、提交或推送。获得授权后只暂存明确路径，并执行 `git diff --cached --check` 与 `git status --short`。
- 禁止提交 SQLite/WAL/SHM、上传数据、秘密、`.next`、`next-env.d.ts`、日志和临时验收产物。
- 默认同时推送 GitHub `origin` 和 Gitee `gitee`，除非用户明确只要求一个远端；推送后分别确认两个远端分支指向目标提交。
- 达到 `P10`、`P20`、`P30` 或进入下一组十个任务时提醒切换新任务，但编号不重置。
