# backend/AGENTS.md

## 适用范围

## 后端新增功能说明

新增接口按 `models -> repository -> handlers -> routes` 顺序实现：模型定义数据契约，repository 负责 SQLite 持久化，handler 负责鉴权、参数校验和响应，routes 负责路径及权限中间件绑定。涉及菜单时同步 `repository/sqlite_store.go`，涉及动作权限时同步 `permissions/actions.go`；前端请求封装放在 `frontend/src/services/`，页面挂载和状态编排放在 `frontend/app/page.tsx` 与 `frontend/src/features/workspace/`。

本文件适用于 `backend/` 及其所有子目录，并补充仓库根目录的 `AGENTS.md`。

## 目录职责

本目录是采集数据管理平台的 Go 后端，使用 Go 1.26、Gin 和 SQLite，默认监听 `:8080`。

- `main.go`：程序入口与依赖组装，负责加载配置、打开数据库、迁移和初始化数据、补录上传文件并启动 Gin。
- `config/`：环境变量和默认配置。
- `database/`：SQLite 连接初始化及连接级设置。
- `handlers/`：所有 HTTP handler，按认证、数据点、用户、部门、角色、菜单、文章和文件拆分为独立 Go 文件。
- `repository/`：当前生产路径使用的 SQLite 持久化实现，包含迁移、HuaJian 组织种子、部门/角色/个人菜单权限和业务 CRUD。
- `store/`：保留的内存存储实现；`main.go` 当前不使用它，不要把新功能只实现于此。
- `routes/`：所有路由注册；`routes.go` 负责总装配，各业务文件负责本域的路径和权限绑定。
- `middleware/`：CORS、会话认证和菜单权限中间件。
- `auth/`：bcrypt 密码处理和 HttpOnly Cookie 会话服务；HTTP 登录、会话查询和登出入口位于 `handlers/auth.go`。
- `permissions/`：动作权限目录及 `resource.action` 稳定编码，是前后端动作权限契约的后端权威来源。
- `verification/`：密码修改验证码服务，使用 Redis 保存一次性验证码并通过 SMTP 发送邮件。
- `models/`：领域模型及请求/响应结构。
- `utils/`：handler 复用的无状态小工具，例如路径 ID 解析、布尔值解析、文件名清理和管理员判断。
- `data/`：SQLite 业务数据库目录；`uploads/`：用户上传文件目录。

## 启动与持久化

- `main.go` 的生产装配使用 `repository.NewSQLiteStore`；`store/store.go` 是遗留内存实现，不能作为生产功能的唯一实现。
- `database.Open` 使用纯 Go `modernc.org/sqlite` 驱动，开启 `foreign_keys`、WAL 和 5 秒 busy timeout，并将最大连接数设为 1。不要引入依赖系统 SQLite 的隐式假设。
- `MigrateAndSeed` 是增量、幂等迁移入口，顺序为预检 → 建表/补列/索引 → 应用菜单 → 部门 → 角色 → 默认账号 → 遗留角色对齐 → `MH` 不变量。遇到碰撞会主动失败并保留数据，不能用清库绕过。
- 启动时 `ReconcileUploadFiles` 会为上传目录中未记录的物理文件补数据库记录；因此测试服务必须同时隔离 SQLite 和上传目录。
- SQLite 主要表包括 `users`、`departments`、`roles`、`menus`、三类菜单关联表、`user_action_permissions`、`sessions`、`articles`、`files` 与 `data_points`。修改列时同步检查所有 `SELECT` 扫描顺序。

## 接口与实现约定

- 健康检查使用 `GET /health`；API 统一位于 `/api`。
- `/api/auth/*` 负责登录会话，其余 API 需要认证；受控业务接口还应保持菜单权限和动作权限检查。
- JSON 字段使用 camelCase，错误信息和用户可见文案默认使用简体中文。
- 保持现有直接清晰的 `handler -> repository` 结构；只有确有复用或复杂业务规则时才新增层次。
- 新增业务接口时，将 HTTP 处理放入 `handlers/<domain>.go`，将路由放入 `routes/<domain>.go`；跨 handler 的无状态工具才放入 `utils/`。
- 修改模型、路由或权限时，检查所有 handler、SQLite repository、前端调用方和 `README.md` 是否需要同步。
- 文件删除默认是可恢复的软删除；未经用户明确授权，不得实现或执行永久物理清理。

## 鉴权层次

- `RequireAuth` 从 Cookie 会话恢复用户，并再次检查 `status` 与 `canLogin`；停用用户的既有会话会立即失效。
- `RequireMenu(store, code)` 校验有效菜单，`RequireAction(store, code)` 校验动作编码；多数业务路由需要二者同时通过。
- `GET /api/menus` 是已登录用户恢复导航树的引导接口，不额外要求菜单动作；不要随意给它加上会导致无法恢复导航的循环依赖。
- 用户、部门、角色的写接口有些只挂动作权限而不挂目标菜单，这是为了允许管理员修复权限配置。改变路由中间件前必须运行路由权限测试。
- `RequireAdmin` 只接受 `super-admin` 或 `system-admin`；只有 `super-admin` 能跨越超级管理员边界。所有判断必须使用 `roleCode`。
- 非管理员角色默认动作来自 `permissions.DefaultRoleCodes()`，即查询/查看动作；个人附加动作保存在 `user_action_permissions`。管理员动作固定为全量，不能个人修改。

## 数据不变量与业务边界

- `MH` 必须且只能存在一个，固定属于 HuaJian 根部门和 `super-admin`，保持在岗、可登录且拥有全部菜单；不得删除、降级或迁移到其他部门。
- 内置管理员角色必须保持启用并拥有全部菜单；角色编码创建后不可修改。角色显示名变化时，兼容字段 `users.role` 要在同一事务中同步。
- 根部门 `huajian` 不可改编码、停用、设置上级或缩减菜单；部门改名时兼容字段 `users.department` 要同步。
- 部门树禁止自引用和循环；删除部门前必须没有下级部门与直属用户，删除角色前必须没有关联用户。
- 有效菜单只合并启用的部门/角色授权和个人附加授权，并补齐祖先节点；停用角色或部门不再贡献权限。
- 文章/文件读取允许公开资源、所有者或管理员；写入、软删除、恢复和永久删除还要求所有者或管理员。不要仅凭动作权限绕过所有权检查。
- 文件上传上限为 32 MiB，服务端存储名必须安全生成；下载、预览和删除通过文件 ID 解析数据库记录，禁止接受任意磁盘路径。
- `DELETE /api/files/:id` 只设置 `deleted_at`；只有回收站中的文件才能调用永久删除并移除物理文件。

## 配置与外部依赖

主要环境变量及默认值：

- `SQLITE_PATH=data/app.db`
- `UPLOAD_DIR=uploads`
- `SERVER_ADDRESS=:8080`
- `CORS_ALLOWED_ORIGINS=*`（开发默认回显任意请求 Origin；生产环境应覆盖为明确白名单）
- `COOKIE_SAMESITE=Lax`
- `COOKIE_SECURE=false`
- `SESSION_COOKIE_NAME=sessionId`
- `SESSION_TTL_HOURS=8`
- `REDIS_ADDR=localhost:6379`
- `REDIS_PASSWORD=`
- `REDIS_DB=0`
- `EMAIL_CONFIG_PATH=~/Desktop/email.txt`（Windows 通常对应 `%USERPROFILE%\Desktop\email.txt`；无法解析用户目录时回退为当前目录 `email.txt`）
- `PASSWORD_CODE_TTL_SECONDS=180`

邮箱文件按 `KEY=VALUE` 读取：`EMAIL_HOST`、`EMAIL_PORT`、`EMAIL_SECURE`、`EMAIL_USER`、`EMAIL_PASS`、`EMAIL_FROM`；`EMAIL_FROM` 空缺时使用 `EMAIL_USER`。密码验证码当前必须写入 Redis，没有内存降级；Redis 或 SMTP 不可用时接口应返回错误，不能把验证码打印到日志或响应。

生产跨站部署通常需要 `COOKIE_SAMESITE=None`、`COOKIE_SECURE=true` 和明确的 `CORS_ALLOWED_ORIGINS`。开发通配符会回显实际 Origin，因为携带凭证时不能返回字面量 `*`。

## 开发与验证

```powershell
cd backend
gofmt -w <修改的.go文件>
go test ./...
go vet ./...
go run .
```

依赖确实变化时再执行 `go mod tidy`，并同时检查 `go.mod`、`go.sum` 差异；不要把它作为每次任务的无条件步骤。

新增路由、权限或持久化行为时，应补充对应层级测试：纯权限目录放在 `permissions/`，存储/迁移放在 `repository/`，HTTP 契约和越权边界放在 `routes/`。测试必须使用 `t.TempDir()` 或等价的独立临时目录，并通过配置指向临时 SQLite 数据库和上传目录。

可按范围先运行聚焦测试，但交付后端改动前仍执行：

```powershell
go test ./...
go vet ./...
```

## 数据保护

`data/` 中的 SQLite、WAL、SHM 文件和 `uploads/` 中的内容均视为用户业务数据。开发、构建、测试和浏览器验收不得删除、覆盖、重置或清空这些内容，也不得通过真实业务 API 批量清理验收数据。

浏览器联调后端必须显式设置隔离路径，例如：

```powershell
$env:SQLITE_PATH="D:\agent\.workspace-temp\<task>\app.db"
$env:UPLOAD_DIR="D:\agent\.workspace-temp\<task>\uploads"
go run .
```

永久删除、迁移碰撞处理、生产数据库修复与上传目录清理都需要用户明确授权；诊断时优先只读查询和复制到隔离环境复现。
