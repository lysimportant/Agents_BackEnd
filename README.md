# 采集数据平台

最小可运行版本：

- `backend`: Go + Gin + SQLite API 服务
- `frontend`: Next.js + TypeScript 企业后台管理界面
- `Socket 客服`: WebSocket 实时会话、聊天监控、图片/文件/表情发送，以及可嵌入第三方网站的悬浮客服组件

## 启动后端

```powershell
cd backend
go mod tidy
go run .
```

后端默认运行在 `http://localhost:8080`，SQLite 数据库默认保存在 `backend/data/app.db`，普通上传文件默认保存在 `backend/uploads/`，客服聊天附件独立保存在 `backend/uploads/socket/<会话ID>/`。

首次启动会幂等创建表、索引和初始数据；后续启动不会覆盖已有业务数据。预置管理员 `MH/123` 仅在账号不存在时写入，密码使用 bcrypt 哈希保存。

后端环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SQLITE_PATH` | `data/app.db` | SQLite 文件路径 |
| `UPLOAD_DIR` | `uploads` | 上传文件目录 |
| `SERVER_ADDRESS` | `:8080` | HTTP 监听地址 |
| `CORS_ALLOWED_ORIGINS` | `*` | 开发默认接受任意 Origin；凭证模式下响应会回显请求的实际 Origin，而不是返回字面量 `*`。生产环境应覆盖为逗号分隔的明确域名 |
| `COOKIE_SAMESITE` | `Lax` | 可选 `Lax`、`Strict`、`None` |
| `COOKIE_SECURE` | `false` | HTTPS 跨站部署时设为 `true` |
| `SESSION_COOKIE_NAME` | `sessionId` | 会话 Cookie 名称 |
| `SESSION_TTL_HOURS` | `8` | 会话有效小时数 |

跨站前后端部署通常需要同时设置 `COOKIE_SAMESITE=None`、`COOKIE_SECURE=true`。开发默认的 `CORS_ALLOWED_ORIGINS=*` 会回显请求的实际 Origin，以兼容携带凭证的请求和任意本地前端端口；公网生产环境应将其覆盖为明确 Origin 白名单。允许来源响应会包含 `Access-Control-Allow-Credentials: true` 和 `Vary: Origin`。

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

除 `GET /health`、`/api/auth/*` 和访客客服入口 `/api/socket/customer*` 外，`/api` 下接口都需要先登录并携带后端写入的 HttpOnly Cookie。访客客服使用服务端生成的随机会话 ID 与访客令牌，令牌只以哈希形式持久化。前端管理请求默认使用 `credentials: 'include'`。

### 认证接口

预置管理员账号：`MH`，初始密码：`123`。密码使用 bcrypt 哈希存储，API 响应不会返回 `password` 或 `passwordHash`。

- `POST /api/auth/login`: 登录并创建会话，Body 示例：`{"username":"MH","password":"123"}`
- `GET /api/auth/session`: 校验并恢复当前会话
- `POST /api/auth/logout`: 退出登录并清除会话

登录和会话响应中的 `user` 会返回当前用户的角色、部门、状态、个人资料及 `actionPermissions` 动作权限数组。`status` 为 `停用` 或 `canLogin=false` 的用户不能登录；用户被停用后，已有会话也会立即失效。恢复为在岗状态时仍需由管理员明确设置 `canLogin=true`。

### 动作权限与按钮

动作权限使用稳定的 `resource.action` 编码，前端应按 `actionPermissions` 控制 CRUD 按钮，后端仍会对每个接口独立校验，不能只依赖按钮隐藏：

- 查询/查看：`dashboard.query|view`、`users.query|view`、`departments.query|view`、`roles.query|view`、`menus.query|view`、`articles.query|view`、`files.query|view`、`socket.query|view`。
- 写动作：`dashboard.create`；各管理资源的 `create`、`update`、`delete`；以及 `users.permissions.update`、`departments.permissions.update`、`roles.permissions.update`、`files.restore`、`files.permanent-delete`、`socket.send`。
- `roleCode=super-admin`（超级管理员）与 `roleCode=system-admin`（系统管理员）固定拥有全部当前动作；其他角色默认只有查询/查看动作，管理员可再为普通用户追加个人动作权限。
- 超级管理员是最高保护角色；只有超级管理员可以创建或分配超级管理员，系统管理员和其他角色不能创建、修改、删除或降级超级管理员。除本人维护 `/api/profile` 外，受控 CRUD、权限配置、文件恢复及彻底删除由动作权限决定。

### 基础接口

- `GET /health`: 健康检查
- `GET /api/data-points`: 获取采集数据
- `POST /api/data-points`: 新增采集数据

### 用户管理

- `GET /api/users`: 获取用户列表
- `POST /api/users`: 新增用户
- `PUT /api/users/:id`: 更新用户
- `DELETE /api/users/:id`: 删除用户

用户 JSON 字段：`username`、`name`、`roleId`、`role`、`roleCode`、`departmentId`、`department`、`status`、`shift`、`phone`、`email`、`age`、`description`、`avatarUrl`、`canLogin`、`password`。`roleId`、`departmentId` 分别关联角色和部门；`role`、`department` 作为兼容名称字段保留，`roleCode` 是只读的安全标识。

说明：

- 新增用户时 `password` 必填，后端会写入 bcrypt 哈希。
- 编辑用户时密码表单不回显，`password` 留空表示不修改原密码。
- `status=停用` 会强制关闭 `canLogin` 并使已有会话失效；登录与会话恢复都会再次校验账号状态。

### 个人资料

- `GET /api/profile`: 获取当前登录用户资料
- `PUT /api/profile`: 更新当前登录用户资料
- `GET /api/users/:id/profile`: 本人或管理员获取指定用户资料
- `PUT /api/users/:id/profile`: 本人或管理员更新指定用户资料

资料更新 Body 可包含 `name`、`email`、`phone`、`age`、`description`、`avatarUrl`。该接口不会修改账号、密码、角色、部门、状态或登录权限；`age` 允许 `0` 到 `150`。

### 菜单管理

- `GET /api/menus`: 获取当前用户的有效菜单（直属部门、角色和个人附加权限的并集，并自动包含已授权子菜单的所有父级）
- `POST /api/menus`: 新增菜单
- `PUT /api/menus/:id`: 更新菜单
- `DELETE /api/menus/:id`: 删除菜单

菜单 JSON 字段：`name`、`code`、`path`、`icon`、`parentId`、`sort`、`status`。

“工作台”是一级分组，默认包含“预览台”（`dashboard`）和“Socket 客服”（`socket-support`）两个二级菜单。已有 `dashboard` 菜单 ID 和授权关系会保留，迁移只补充其工作台父级。

### 用户菜单权限

- `GET /api/users/:id/menus`: 查询用户个人附加菜单
- `PUT /api/users/:id/menus`: 保存用户个人附加菜单，Body 示例：`{"menuIds":[1,2,3]}`
- `GET /api/users/:id/permissions`: 查询权限明细，返回 `departmentMenuIds`、`roleMenuIds`、`userMenuIds`、`effectiveMenuIds`、`roleActionCodes`、`userActionCodes` 和 `effectiveActionCodes`
- `PUT /api/users/:id/actions`: 超级管理员或系统管理员保存普通用户的个人按钮/动作权限，Body 与响应均为 `{"actionCodes":["articles.create","files.update"]}`；传空数组可清空个人授权

### 部门管理

- `GET /api/departments`: 获取按 `parentId` 组织的部门列表
- `GET /api/departments/:id`: 获取部门详情
- `POST /api/departments`: 新增部门
- `PUT /api/departments/:id`: 更新部门
- `DELETE /api/departments/:id`: 删除没有下级部门和用户的部门
- `GET /api/departments/:id/menus`: 查询部门直接分配的菜单
- `PUT /api/departments/:id/menus`: 保存部门菜单权限，Body 示例：`{"menuIds":[1,2,3]}`
- `GET /api/departments/:id/users`: 查询直属该部门的用户，直接返回用户数组

### 角色管理

- `GET /api/roles`: 获取角色列表
- `GET /api/roles/:id`: 获取角色详情
- `POST /api/roles`: 新增角色
- `PUT /api/roles/:id`: 更新角色
- `DELETE /api/roles/:id`: 删除没有关联用户的非系统角色
- `GET /api/roles/:id/menus`: 查询角色直接权限，响应示例：`{"menuIds":[1,2,3]}`
- `PUT /api/roles/:id/menus`: 保存角色菜单权限，Body 与响应均为 `{"menuIds":[1,2,3]}`
- `GET /api/roles/:id/users`: 查询使用该角色的用户，直接返回用户数组

角色 JSON 字段：`name`、`code`、`description`、`sort`、`status`。系统幂等创建 11 个常见及购物预留角色：`super-admin`（超级管理员）、`system-admin`（系统管理员）、`department-admin`（部门管理员）、`content-editor`（内容编辑）、`auditor`（审核员）、`viewer`（普通用户）、`product-manager`（商品管理员）、`order-manager`（订单管理员）、`warehouse-manager`（仓库管理员）、`customer-service`（客服专员）和 `finance`（财务人员）。旧内置 `operations-admin` 会安全迁移为部门管理员，关联用户和权限均保留；自定义角色不会被启动迁移删除。购物角色目前是预留角色，默认只有工作台和查询/查看动作，待商品、订单、库存等菜单与 API 接入后再配置对应权限。非管理员角色及普通部门默认具有工作台权限，超级管理员、系统管理员、根部门和 `board-office` 保留全部菜单权限。角色编码创建后不可在前端修改，修改显示名称时关联用户名称会在同一事务中同步更新。

用户的有效菜单是启用状态直属部门、启用状态角色与个人附加菜单的并集，停用部门或角色不再贡献菜单权限。HuaJian 组织结构作为幂等初始数据写入；`MH` 会关联根部门和 `super-admin` 角色，两者始终补齐全部菜单权限。升级时只将 `MH` 迁入超级管理员，其他既有系统管理员保持 `system-admin`，不会被批量提升。启动迁移不会清空已有菜单、部门/角色/个人权限或业务数据，也不会重置已有 `MH` 的密码。

默认管理员 `MH` 不可删除；通过用户接口编辑时会强制保留 `MH` 账号、`超级管理员` 角色、根部门归属和可登录状态，但仍可更新姓名、联系方式及密码。超级管理员可以将超级管理员角色分配给其他账号，其他角色不能分配或调整超级管理员；根部门权限不可缩减。

用户、部门、角色和菜单接口同时按有效菜单与动作编码鉴权。超级管理员和系统管理员固定拥有全部当前动作，且个人动作权限不可修改；其他角色默认只有查询、查看动作，其有效权限是角色动作与管理员授予的个人动作并集。普通用户不能自行提权。只有超级管理员可以创建或调整超级管理员、系统管理员；系统管理员不能操作超级管理员、`MH` 或管理员角色边界。所有限制使用稳定 `roleCode` 校验，不依赖可编辑的角色名称。

### 文章管理

- `GET /api/articles`: 获取文章列表
- `GET /api/articles/:id`: 获取文章详情
- `POST /api/articles`: 新增文章
- `PUT /api/articles/:id`: 更新文章
- `DELETE /api/articles/:id`: 删除文章
- `GET /api/articles/export?format=csv|pdf`: 导出当前用户可见文章；CSV 使用 UTF-8 BOM 并防止公式注入，PDF 在内存中生成

文章查询、详情和导出可由具有文章菜单与相应动作的角色使用；新增、修改和删除按动作权限控制。

文章 JSON 字段：`title`、`category`、`author`、`status`、`summary`、`content`。状态可使用 `已发布`、`草稿`、`待审核`。

前端还可将单篇文章导出为 Excel 兼容 CSV、打印/PDF、Word、分页 PNG、Markdown 或带 `Article` 结构化数据的 SEO HTML。Markdown 会根据正文标题自动生成目录和显式锚点，正文没有标题时不生成目录；重复标题会生成唯一锚点。公开且已发布的文章会输出文章语义信息，私密或未发布文章不会输出可索引标记。

### 文件管理

- `GET /api/files`: 获取文件元数据列表
- `GET /api/files/:id`: 获取文件元数据详情
- `POST /api/files`: 上传文件，`multipart/form-data` 字段为 `file`、`displayName`、`category`、`description`
- `PUT /api/files/:id`: 更新文件元数据，JSON 字段为 `displayName`、`category`、`description`
- `GET /api/files/:id/download`: 下载文件内容
- `DELETE /api/files/:id`: 将文件移入回收站（软删除，保留物理文件）
- `POST /api/files/:id/restore`: 从回收站恢复文件

文件查询、详情、预览和下载可由具有文件菜单与相应动作的角色使用；上传、修改、软删除、恢复和彻底删除按动作权限控制。

文件安全约束：

- 单文件上传限制为 32MB。
- 上传后使用随机服务端存储名，API 不返回绝对路径或存储路径。
- 原始文件名通过 `filepath.Base` 清理，下载和删除只按文件 ID 查询元数据。
- 删除默认采用可恢复软删除：文件移入回收站但物理上传内容保留，直到用户明确授权永久清理。
- 服务端会校验存储名和最终路径，防止路径穿越。
- 公开图片在管理界面中带描述性替代文本与 `ImageObject` 语义；私密图片不会输出该索引标记。真正面向搜索引擎公开收录时仍需部署无需登录的公开详情 URL。

### Socket 在线客服

管理端“工作台 → Socket 客服”会实时列出全部客服会话，显示访客在线状态、最近消息和消息数量；选择会话后可监视完整聊天记录。具有 `socket.send` 动作权限的客服还可以回复文字、表情、图片和文件。管理端连接与历史接口需要 `socket-support` 菜单及相应动作权限：

- `GET /api/socket/admin`: 管理端 WebSocket，推送客户上线、离线和新消息
- `GET /api/socket/conversations`: 获取全部客服会话
- `GET /api/socket/conversations/:id/messages`: 获取指定会话历史消息
- `POST /api/socket/conversations/:id/messages`: 发送文字或表情，Body：`{"messageType":"text","content":"您好"}`
- `POST /api/socket/conversations/:id/files`: 发送图片或文件，`multipart/form-data` 字段为 `file`
- `GET /api/socket/conversations/:id/files/:messageId`: 管理端预览附件；增加 `?download=1` 下载

访客组件使用以下公开接口：

- `GET /api/socket/customer`: 访客 WebSocket；首次连接会返回随机会话 ID 和访客令牌，重连时携带二者
- `POST /api/socket/customer/:id/files`: 访客发送图片或文件，请求头携带 `X-Socket-Visitor-Token`
- `GET /api/socket/customer/:id/files/:messageId`: 访客读取本会话附件，请求头携带 `X-Socket-Visitor-Token`

可复用悬浮客服组件位于 `frontend/public/socket/socket-customer-widget.js`，API 等公共参数位于 `frontend/public/socket/socket-config.js`。将下面脚本加入任意网站，右下角会出现客服按钮；访客首次点击并连接后，会话 ID 会自动登记到管理端 Socket 客服页面，访客页面 URL 不需要携带 API 参数：

```html
<script src="http://localhost:3000/socket/socket-config.js"></script>
<script
  src="http://localhost:3000/socket/socket-customer-widget.js"
  data-title="在线客服"
  data-color="#1677ff"
  data-position="right"
  data-session-key="default"
></script>
```

`socket-config.js` 中的 `apiBase` 是统一后端配置，本地默认为 `http://localhost:8000`。脚本同时暴露 `window.SocketCustomerWidget.mount(options)`，可在 SPA、CMS 或微前端中手动挂载多个定制实例。同一浏览器默认复用同一个访客会话；为不同站点、账号或窗口设置不同 `data-session-key` / `options.sessionKey`，即可在同一电脑创建多个独立客服会话。管理端可同时接收和切换查看任意数量的访客会话。生产环境应把配置文件中的 `apiBase` 和脚本地址改成实际 HTTPS 地址，并在 `CORS_ALLOWED_ORIGINS` 中明确允许嵌入站点来源。客服附件限制为 32 MiB，并独立存储在 `UPLOAD_DIR/socket/` 分类目录中。

示例请求：

```powershell
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
Invoke-RestMethod -Uri http://localhost:8080/api/auth/login -Method Post -ContentType 'application/json' -Body '{"username":"MH","password":"123"}' -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/articles -Method Post -ContentType 'application/json' -Body '{"title":"生产日报","category":"通知公告","author":"管理员","status":"草稿","summary":"今日生产摘要","content":"正文内容"}' -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/files -Method Post -Form @{file=Get-Item .\README.md; displayName='README'; category='文档'; description='项目说明'} -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/auth/logout -Method Post -WebSession $session
```
