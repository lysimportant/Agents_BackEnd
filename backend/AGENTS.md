# backend/AGENTS.md

## 适用范围

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
- `models/`：领域模型及请求/响应结构。
- `utils/`：handler 复用的无状态小工具，例如路径 ID 解析、布尔值解析、文件名清理和管理员判断。
- `data/`：SQLite 业务数据库目录；`uploads/`：用户上传文件目录。

## 接口与实现约定

- 健康检查使用 `GET /health`；API 统一位于 `/api`。
- `/api/auth/*` 负责登录会话，其余 API 需要认证；受控业务接口还应保持菜单权限检查。
- JSON 字段使用 camelCase，错误信息和用户可见文案默认使用简体中文。
- 保持现有直接清晰的 `handler -> repository` 结构；只有确有复用或复杂业务规则时才新增层次。
- 新增业务接口时，将 HTTP 处理放入 `handlers/<domain>.go`，将路由放入 `routes/<domain>.go`；跨 handler 的无状态工具才放入 `utils/`。
- 修改模型、路由或权限时，检查所有 handler、SQLite repository、前端调用方和 `README.md` 是否需要同步。
- 文件删除默认是可恢复的软删除；未经用户明确授权，不得实现或执行永久物理清理。

主要环境变量及默认值：

- `SQLITE_PATH=data/app.db`
- `UPLOAD_DIR=uploads`
- `SERVER_ADDRESS=:8080`
- `CORS_ALLOWED_ORIGINS=*`（开发默认回显任意请求 Origin；生产环境应覆盖为明确白名单）
- `COOKIE_SAMESITE=Lax`
- `COOKIE_SECURE=false`
- `SESSION_COOKIE_NAME=sessionId`
- `SESSION_TTL_HOURS=8`

## 开发与验证

```powershell
cd backend
go mod tidy
gofmt -w <修改的.go文件>
go test ./...
go vet ./...
go run .
```

新增路由、权限或持久化行为时，应补充相应测试。测试必须使用 `t.TempDir()` 或等价的独立临时目录，并通过配置指向临时 SQLite 数据库和上传目录。

## 数据保护

`data/` 中的 SQLite、WAL、SHM 文件和 `uploads/` 中的内容均视为用户业务数据。开发、构建、测试和浏览器验收不得删除、覆盖、重置或清空这些内容，也不得通过真实业务 API 批量清理验收数据。
