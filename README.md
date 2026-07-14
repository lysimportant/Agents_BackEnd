# 采集数据平台

最小可运行版本：

- `backend`: Go + Gin + SQLite API 服务
- `frontend`: Next.js + TypeScript 若依式后台管理界面

## 启动后端

```powershell
cd backend
go mod tidy
go run .
```

后端默认运行在 `http://localhost:8080`，SQLite 数据库默认保存在 `backend/data/app.db`，上传文件默认保存在 `backend/uploads/`。

首次启动会幂等创建表、索引和初始数据；后续启动不会覆盖已有业务数据。预置管理员 `MH/123` 仅在账号不存在时写入，密码使用 bcrypt 哈希保存。

后端环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SQLITE_PATH` | `data/app.db` | SQLite 文件路径 |
| `UPLOAD_DIR` | `uploads` | 上传文件目录 |
| `SERVER_ADDRESS` | `:8080` | HTTP 监听地址 |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | 允许携带凭证的来源白名单，多个来源用逗号分隔；不接受 `*` |
| `COOKIE_SAMESITE` | `Lax` | 可选 `Lax`、`Strict`、`None` |
| `COOKIE_SECURE` | `false` | HTTPS 跨站部署时设为 `true` |
| `SESSION_COOKIE_NAME` | `sessionId` | 会话 Cookie 名称 |
| `SESSION_TTL_HOURS` | `8` | 会话有效小时数 |

跨站前后端部署通常需要同时设置 `COOKIE_SAMESITE=None`、`COOKIE_SECURE=true`，并将前端完整 Origin 加入 `CORS_ALLOWED_ORIGINS`。凭证模式下不会使用通配符来源，允许来源响应会包含 `Access-Control-Allow-Credentials: true` 和 `Vary: Origin`。

## 启动前端

```powershell
cd frontend
npm install
npm run dev
```

前端默认运行在 `http://localhost:3000`，默认调用后端 `http://localhost:8080`。

如需覆盖后端地址，可在前端环境变量中设置：

```powershell
$env:NEXT_PUBLIC_API_BASE_URL="http://localhost:8080"
npm run dev
```

## API

除 `GET /health` 和 `/api/auth/*` 外，`/api` 下接口都需要先登录并携带后端写入的 HttpOnly Cookie。前端请求默认使用 `credentials: 'include'`。

### 认证接口

预置管理员账号：`MH`，初始密码：`123`。密码使用 bcrypt 哈希存储，API 响应不会返回 `password` 或 `passwordHash`。

- `POST /api/auth/login`: 登录并创建会话，Body 示例：`{"username":"MH","password":"123"}`
- `GET /api/auth/session`: 校验并恢复当前会话
- `POST /api/auth/logout`: 退出登录并清除会话

### 基础接口

- `GET /health`: 健康检查
- `GET /api/data-points`: 获取采集数据
- `POST /api/data-points`: 新增采集数据

### 用户管理

- `GET /api/users`: 获取用户列表
- `POST /api/users`: 新增用户
- `PUT /api/users/:id`: 更新用户
- `DELETE /api/users/:id`: 删除用户

用户 JSON 字段：`username`、`name`、`role`、`department`、`status`、`shift`、`phone`、`email`、`password`。

说明：

- 新增用户时 `password` 必填，后端会写入 bcrypt 哈希。
- 编辑用户时密码表单不回显，`password` 留空表示不修改原密码。

### 菜单管理

- `GET /api/menus`: 获取菜单列表
- `POST /api/menus`: 新增菜单
- `PUT /api/menus/:id`: 更新菜单
- `DELETE /api/menus/:id`: 删除菜单

菜单 JSON 字段：`name`、`code`、`path`、`icon`、`parentId`、`sort`、`status`。

### 用户菜单权限

- `GET /api/users/:id/menus`: 查询用户已分配菜单
- `PUT /api/users/:id/menus`: 保存用户菜单权限，Body 示例：`{"menuIds":[1,2,3]}`

### 文章管理

- `GET /api/articles`: 获取文章列表
- `GET /api/articles/:id`: 获取文章详情
- `POST /api/articles`: 新增文章
- `PUT /api/articles/:id`: 更新文章
- `DELETE /api/articles/:id`: 删除文章

文章 JSON 字段：`title`、`category`、`author`、`status`、`summary`、`content`。状态可使用 `已发布`、`草稿`、`待审核`。

### 文件管理

- `GET /api/files`: 获取文件元数据列表
- `GET /api/files/:id`: 获取文件元数据详情
- `POST /api/files`: 上传文件，`multipart/form-data` 字段为 `file`、`displayName`、`category`、`description`
- `PUT /api/files/:id`: 更新文件元数据，JSON 字段为 `displayName`、`category`、`description`
- `GET /api/files/:id/download`: 下载文件内容
- `DELETE /api/files/:id`: 将文件移入回收站（软删除，保留物理文件）
- `POST /api/files/:id/restore`: 从回收站恢复文件

文件安全约束：

- 单文件上传限制为 10MB。
- 上传后使用随机服务端存储名，API 不返回绝对路径或存储路径。
- 原始文件名通过 `filepath.Base` 清理，下载和删除只按文件 ID 查询元数据。
- 删除默认采用可恢复软删除：文件移入回收站但物理上传内容保留，直到用户明确授权永久清理。
- 服务端会校验存储名和最终路径，防止路径穿越。

示例请求：

```powershell
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
Invoke-RestMethod -Uri http://localhost:8080/api/auth/login -Method Post -ContentType 'application/json' -Body '{"username":"MH","password":"123"}' -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/articles -Method Post -ContentType 'application/json' -Body '{"title":"生产日报","category":"通知公告","author":"管理员","status":"草稿","summary":"今日生产摘要","content":"正文内容"}' -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/files -Method Post -Form @{file=Get-Item .\README.md; displayName='README'; category='文档'; description='项目说明'} -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/auth/logout -Method Post -WebSession $session
```
