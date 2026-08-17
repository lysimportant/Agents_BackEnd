# 采集数据平台

第一次运行或需要了解后台操作流程，请阅读[平台操作说明](./docs/operations-guide.md)。

最小可运行版本：

- `backend`: Go + Gin + SQLite API 服务
- `frontend`: Next.js + TypeScript 企业后台管理界面
- `Socket 客服`: WebSocket 实时会话、聊天监控、图片/文件/表情发送，以及可嵌入第三方网站的悬浮客服组件
- `portal`: Next.js + TypeScript + Tailwind 的 C 端公开内容门户，仅通过后端 `/api/public/*` 只读接口展示已发布内容

## 启动后端

```powershell
cd backend
go mod tidy
go run .
```

后端默认运行在 `http://localhost:8080`，SQLite 数据库默认保存在 `backend/data/app.db`，普通上传文件默认保存在 `backend/uploads/`，客服聊天附件独立保存在 `backend/uploads/socket/<会话ID>/`。

首次启动会幂等创建表、索引和初始数据；仅当用户表为空时创建初始账号，后续启动不会恢复或重建已修改、删除的账号。密码使用 bcrypt 哈希保存。

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
| `VISITOR_LOG_RETENTION_DAYS` | `90` | 访问分析日志保留天数；设为 `0` 可关闭自动清理 |
| `HOST_AGENT_TOKEN` | 空 | 宿主机代理共享令牌；留空时禁用部署机直连，生产值至少使用 32 字节随机内容 |

Docker 启动时默认读取仓库根目录的 `email.txt`，并将其只读挂载为容器内的 `/app/email.txt`；可通过 `EMAIL_CONFIG_FILE` 覆盖宿主机文件路径。也可直接设置 `EMAIL_HOST`、`EMAIL_PORT`、`EMAIL_SECURE`、`EMAIL_USER`、`EMAIL_PASS`、`EMAIL_FROM`，非空环境变量优先于文件配置。`email.txt` 已被 Git 忽略，不要把真实邮箱密码提交到仓库。

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

## 启动门户（C 端）

```powershell
cd portal
npm install
npm run dev
```

门户默认运行在 `http://localhost:3001`，默认调用后端 `http://localhost:8080`。
面向普通访问者，展示经 B 端明确发布到门户的文章、图片与资源，支持简体中文 / English / 日本語、三套主题与移动端适配。
门户页面只通过后端 `/api/public/*` 只读接口取数，不携带后台登录 Cookie，也不提供任何写能力。

如需覆盖后端地址或站点地址，可设置 `NEXT_PUBLIC_API_BASE_URL` 与 `NEXT_PUBLIC_SITE_URL`；详见 `portal/README.md`。

门户构建与验证：

```powershell
cd portal
npm run typecheck
npm run lint
npm run build
npm run docs
```

门户只消费后端公开只读接口，具体路由与契约见 `docs/api/openapi.yaml` 与 `docs/portal-requirements.md`。

## Docker Compose 启动

根 `docker-compose.yml` 会同时启动 Redis、后端、管理前端和 C 端门户：

```powershell
Copy-Item .env.example .env
docker compose up --build
```

默认访问地址为后端 `http://localhost:8080`、管理前端 `http://localhost:3000`、C 端门户 `http://localhost:3001`。`BACKEND_PORT`、`FRONTEND_PORT`、`PORTAL_PORT` 控制宿主机映射端口，`BACKEND_INTERNAL_PORT`、`PORTAL_INTERNAL_PORT` 控制对应容器的监听端口。修改门户映射端口时还需同步修改 `PORTAL_SITE_URL` 与 `CORS_ALLOWED_ORIGINS`，修改后端映射端口时需同步修改 `FRONTEND_API_BASE_URL` 与 `PORTAL_API_BASE_URL`；这些公开变量会写入前端构建产物，变更后必须重新构建镜像。

### 公开接口（C 端门户）

以下接口无需登录，仅返回满足门户发布条件的内容，不返回存储路径、所有者或删除信息：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/public/articles` | 公开文章列表，支持 `page`、`pageSize`、`category`、`keyword` |
| GET | `/api/public/articles/:id` | 公开文章详情 |
| GET | `/api/public/images` | 公开图片列表 |
| GET | `/api/public/resources` | 公开资源列表 |
| GET | `/api/public/files/:id/preview` | 公开文件内联预览 |
| GET | `/api/public/files/:id/download` | 公开文件下载 |
| GET | `/api/public/categories` | 公开分类聚合 |
| GET | `/api/public/site-summary` | 站点聚合概览 |
| GET | `/api/public/search?keyword=` | 聚合搜索（文章/图片/资源） |
## API

除 `GET /health`、`/api/auth/*` 和访客客服入口 `/api/socket/customer*` 外，`/api` 下接口都需要先登录并携带后端写入的 HttpOnly Cookie。访客客服使用服务端生成的随机会话 ID 与访客令牌，令牌只以哈希形式持久化。前端管理请求默认使用 `credentials: 'include'`。

### 认证接口

登录账号由初始化数据或超级管理员创建，登录页不会预填任何账号或密码。密码使用 bcrypt 哈希存储，API 响应不会返回 `password` 或 `passwordHash`。

- `POST /api/auth/login`: 登录并创建会话，Body 包含 `username` 和 `password`
- `GET /api/auth/session`: 校验并恢复当前会话
- `POST /api/auth/logout`: 退出登录并清除会话

登录和会话响应中的 `user` 会返回当前用户的角色、部门、状态、个人资料及 `actionPermissions` 动作权限数组。`status` 为 `停用` 或 `canLogin=false` 的用户不能登录；用户被停用后，已有会话也会立即失效。恢复为在岗状态时仍需由管理员明确设置 `canLogin=true`。

### 动作权限与按钮

动作权限使用稳定的 `resource.action` 编码，前端应按 `actionPermissions` 控制 CRUD 按钮，后端仍会对每个接口独立校验，不能只依赖按钮隐藏：

- 查询/查看：`dashboard.query|view`、`users.query|view`、`departments.query|view`、`roles.query|view`、`menus.query|view`、`articles.query|view`、`files.query|view`、`socket.query|view`、`visitor-analytics.query|view`。
- 写动作：`dashboard.create`；各管理资源的 `create`、`update`、`delete`；以及 `users.permissions.update`、`departments.permissions.update`、`roles.permissions.update`、`files.restore`、`files.permanent-delete`、`socket.send`、`socket.delete`。
- `roleCode=super-admin`（超级管理员）与 `roleCode=system-admin`（系统管理员）固定拥有全部当前动作；其他角色默认只有查询/查看动作，管理员可再为普通用户追加个人动作权限。
- 超级管理员是最高保护角色；只有超级管理员可以创建或分配超级管理员，系统管理员和其他角色不能创建、修改、删除或降级超级管理员。除本人维护 `/api/profile` 外，受控 CRUD、权限配置、文件恢复及彻底删除由动作权限决定。

### 基础接口

- `GET /health`: 健康检查
- `GET /api/data-points`: 获取采集数据
- `POST /api/data-points`: 新增采集数据
- `GET /api/server/metrics`: 获取后端运行环境的 CPU/核心/负载、物理内存与交换区、文件系统与磁盘 I/O、总流量与网卡/连接、硬件温度、后端进程和即时健康告警；工作台默认每 5 秒采样并计算最近 5 分钟的网络及磁盘吞吐趋势。Docker 部署显示容器视角，平台或容器权限不支持的扩展项会通过 `collectionWarnings` 说明
- `GET /api/server/connections`: 点击预览台“活动连接”后按需获取连接详情；返回 TCP/UDP、IPv4/IPv6、本地与远端端点、连接状态、PID 和权限允许时的进程名，最多枚举 5000 个套接字并返回前 500 条诊断明细
- `GET /api/server/terminal`: 所有登录用户均可使用的 SSH WebSocket；全局 Header 可打开同一弹窗中的多终端标签，连接后支持 SFTP 目录逐层进入、返回上级、当前目录搜索、不超过 1 MiB 的 UTF-8 文本编辑保存，以及不超过 10 MiB 的常见图片和 PDF 只读预览。Bash/Zsh 终端执行 `cd`、`cd -`、`pushd` 等目录切换后会通过标准 OSC 7 报告实际工作目录，并自动同步左侧目录。连接密码、私钥和编辑内容仅在当前页面内存中使用，最小化弹窗或切换管理页面时保持，退出登录或关闭浏览器页面后全部清除；首次连接必须确认服务器 SHA256 主机指纹
- `GET /api/server/host-terminal`: 仅超级管理员可使用的部署机直连 WebSocket；通过宿主机代理提供与 SSH 工作区一致的多终端、目录步进、文件编辑及媒体预览，不依赖 SSH 服务或 22 端口
- `GET /api/server/host-agent`: Linux 宿主机代理使用 Bearer 共享令牌主动注册的 WebSocket；一个后端实例同时接受一个代理并复用最多 32 个临时终端会话

### 部署机直连（无需 22 端口）

部署机直连由宿主机上的独立低权限进程执行命令。Docker 后端只负责认证和转发，因此不会把容器 shell 误当成宿主机，也不需要给后端容器挂载 Docker Socket、宿主机根目录或特权模式。

1. 生成共享令牌，并将同一值分别配置到仓库部署 `.env` 的 `HOST_AGENT_TOKEN` 和宿主机 `/etc/collector-host-agent.env`。令牌不得提交到 Git：

```bash
openssl rand -hex 32
```

2. 构建 Linux 宿主机代理。以下命令使用 Docker BuildKit 导出静态二进制，不会启动特权容器：

```bash
docker build --file backend/Dockerfile.host-agent --output type=local,dest=./dist backend
```

3. 创建专用系统账号并安装二进制、环境文件和 systemd 服务：

```bash
sudo useradd --system --create-home --shell /bin/bash collector-terminal
sudo install -m 0755 dist/collector-host-agent /usr/local/bin/collector-host-agent
sudo install -m 0600 deploy/host-agent/collector-host-agent.env.example /etc/collector-host-agent.env
sudo install -m 0644 deploy/host-agent/collector-host-agent.service /etc/systemd/system/collector-host-agent.service
sudo systemctl daemon-reload
sudo systemctl enable --now collector-host-agent
```

`HOST_AGENT_SERVER_URL` 应指向后端可访问的 `ws://` 或 `wss://.../api/server/host-agent`。反向代理必须为 `/api/server/host-agent` 和 `/api/server/host-terminal` 保留 WebSocket `Upgrade`，并关闭过短的读取超时。代理默认从 `/` 启动 shell，但实际命令和文件权限仅等于 `collector-terminal` 系统账号；需要管理权限时应在宿主机用最小范围 sudoers 规则授权，再在终端内显式执行 `sudo`，不要直接以 root 运行代理。

代理令牌只用于代理到后端的注册认证；浏览器不会接触该令牌。部署机直连还会再次校验后台 HttpOnly 会话与不可变 `roleCode=super-admin`，系统管理员和普通用户即使直接请求接口也会收到 403。代理离线时前端会显示明确错误，现有 SSH 模式不受影响。

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
- `PUT /api/users/:id/profile`: 本人或管理员更新指定用户资料；管理员目标仅允许超级管理员修改

资料更新 Body 可包含 `name`、`email`、`phone`、`age`、`description`、`avatarUrl`。该接口会直接持久化到 SQLite，不会修改账号、密码、角色、部门、状态或登录权限；`age` 允许 `0` 到 `150`。超级管理员可以互相修改资料，系统管理员和其他角色不能修改超级管理员资料。

### 菜单管理

- `GET /api/menus`: 获取当前用户的有效菜单（直属部门、角色和个人附加权限的并集，并自动包含已授权子菜单的所有父级）
- `POST /api/menus`: 新增菜单
- `PUT /api/menus/:id`: 更新菜单
- `DELETE /api/menus/:id`: 删除菜单

菜单 JSON 字段：`name`、`code`、`path`、`icon`、`parentId`、`sort`、`status`。

“工作台”是一级分组，默认包含“预览台”（`dashboard`）、“业务资源”（`business-resources`）、“在线聊天”（`socket-support`）和“访问分析”（`visitor-analytics`）四个二级菜单。“业务资源”独立展示用户、菜单、文章的总量、有效量、构成和可用率；已有 `dashboard` 菜单 ID 和授权关系会保留，迁移只补充工作台父级和新增菜单。

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

用户的有效菜单是启用状态直属部门、启用状态角色与个人附加菜单的并集，停用部门或角色不再贡献菜单权限。HuaJian 组织结构作为幂等初始数据写入；管理员角色和根部门始终补齐全部菜单权限。启动迁移不会清空已有菜单、部门/角色/个人权限或业务数据。超级管理员可以将超级管理员角色分配给其他账号，其他角色不能分配或调整超级管理员；根部门权限不可缩减。

用户、部门、角色和菜单接口同时按有效菜单与动作编码鉴权。超级管理员和系统管理员固定拥有全部当前动作，且个人动作权限不可修改；其他角色默认只有查询、查看动作，其有效权限是角色动作与管理员授予的个人动作并集。普通用户不能自行提权。只有超级管理员可以创建或调整超级管理员、系统管理员；系统管理员不能操作超级管理员或管理员角色边界。所有限制使用稳定 `roleCode` 校验，不依赖用户名或可编辑的角色名称。

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

- `GET /api/files`: 获取文件元数据列表；超级管理员还会看到内部聊天和客服聊天附件，统一归类为“聊天数据”且只读
- `GET /api/files/:id`: 获取文件元数据详情
- `POST /api/files`: 上传文件，`multipart/form-data` 字段为 `file`、`displayName`、`category`、`description`
- `PUT /api/files/:id`: 更新文件元数据，JSON 字段为 `displayName`、`category`、`description`
- `GET /api/files/:id/download`: 下载文件内容
- `GET /api/files/chat-data/:source/:id/preview`: 超级管理员预览聊天附件，`source` 为 `internal-chat` 或 `customer-chat`
- `GET /api/files/chat-data/:source/:id/download`: 超级管理员下载聊天附件
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

### 内部聊天

登录用户可通过管理后台右上角的聊天按钮打开独立页面 `http://localhost:3000/chat`。该页面不加载管理后台 Header 和侧栏，采用类似微信的双栏布局；默认进入全员群聊，也可以从左侧用户列表选择在岗且允许登录的用户发起私聊。消息持久化到 SQLite，页面每 2 秒拉取一次当前会话的新状态。

- `GET /api/internal-chat/users`: 获取可私聊用户，并返回最近 15 秒内活跃的 `online` 状态
- `POST /api/internal-chat/presence`: 更新当前用户在线状态
- `GET /api/internal-chat/messages?peerId=0`: 获取全员群聊消息；`peerId` 为用户 ID 时获取双方私聊消息
- `POST /api/internal-chat/attachments`: 以 `multipart/form-data` 的 `file` 字段上传不超过 10 MiB 的图片或常用文档，返回仅归当前用户所有的临时附件 ID
- `POST /api/internal-chat/messages`: 发送消息；Body 示例：`{"recipientId":null,"content":"大家好","attachmentIds":[1]}`，`recipientId=null` 表示群聊；附件 ID 会在同一事务内校验归属并绑定消息
- `GET /api/internal-chat/attachments/:id/preview`: 预览图片附件；仅消息发送者、私聊接收者、群聊成员或管理员可访问
- `GET /api/internal-chat/attachments/:id/download`: 下载附件；鉴权范围与预览接口相同

### 访问分析

管理端“工作台 → 访问分析”仅对具备 `visitor-analytics` 菜单和查询动作的用户开放，默认超级管理员和系统管理员可查看。服务端会记录进入网站/API 的请求摘要，用于全球访问趋势和来源分析：连接 IP、可信代理提供的国家/地区/城市/ISP、Host、方法、路径、状态码、耗时、响应大小、User-Agent、浏览器、系统、设备、Referer、语言，以及已登录用户标识。系统不会记录 Cookie、密码、请求正文或查询参数；地区信息在反向代理未提供可信 Header 时显示“未知”。

- `GET /api/visitor-analytics?range=24h|7d|30d&page=1&pageSize=10`: 返回分页访问明细、请求量、独立 IP、登录访问、异常量、平均耗时、国家/地区排行、热门路径和趋势数据；`pageSize` 支持 `10`、`20`、`30`、`50`、`100`，默认 `10`；可用 `keyword` 搜索 IP、地区、路径或 User-Agent，可用 `statusCode` 过滤状态码。

访问 IP 属于敏感访问元数据，仅管理员可查阅，默认保留 90 天，可通过 `VISITOR_LOG_RETENTION_DAYS` 调整。日志中间件跳过 `OPTIONS` 与 `/health`，并按小时自动清理过期记录。

### Socket 在线客服

管理端“工作台 → Socket 客服”会实时列出全部客服会话，显示会话标题、访客在线状态、最近消息和消息数量，并支持按标题和更新时间范围搜索；选择会话后可监视完整聊天记录，访客端会收到客服接入通知。离线或已关闭的会话仍可查看历史，但不能再接入、回复或发送文件。具有 `socket.send` 动作权限的客服可以回复在线访客，具有 `socket.delete` 权限的人员可直接在会话列表删除，无需先打开。软删除只从列表隐藏会话，聊天记录与附件继续保留。管理端连接与历史接口需要 `socket-support` 菜单及相应动作权限：

独立访客聊天窗口位于 `http://localhost:3000/socket/chat/new`，页面标题为“客服咨询”。首次连接成功后，地址会自动替换为 `/socket/chat/<聊天ID>`，URL 中不携带后端 API 地址。这个完整聊天页面与下方可嵌入其他网站的右下角悬浮组件是两个彼此独立的入口。

- `GET /api/socket/admin`: 管理端 WebSocket，推送客户上线、离线和新消息
- `GET /api/socket/conversations`: 获取全部客服会话
- `GET /api/socket/conversations/:id/messages`: 获取指定会话历史消息
- `POST /api/socket/conversations/:id/messages`: 发送文字或表情，Body：`{"messageType":"text","content":"您好"}`
- `POST /api/socket/conversations/:id/files`: 发送图片或文件，`multipart/form-data` 字段为 `file`
- `POST /api/socket/conversations/:id/join`: 标记客服进入会话，并通知访客端
- `DELETE /api/socket/conversations/:id`: 软删除会话，不物理删除聊天记录与附件
- `GET /api/socket/conversations/:id/files/:messageId`: 管理端预览附件；增加 `?download=1` 下载
- `GET /api/socket/notifications`: 所有已登录用户使用的全局通知 WebSocket，不要求进入 Socket 客服页面

访客组件使用以下公开接口：

- `GET /api/socket/customer`: 访客 WebSocket；首次连接会返回随机会话 ID 和访客令牌，重连时携带二者
- `PUT /api/socket/customer/:id/title`: 访客修改会话标题，请求头携带 `X-Socket-Visitor-Token`
- `DELETE /api/socket/customer/:id`: 访客软删除自己的会话，请求头携带 `X-Socket-Visitor-Token`
- `POST /api/socket/customer/:id/close`: 访客显式结束咨询时关闭会话
- `POST /api/socket/customer/:id/files`: 访客发送图片或文件，请求头携带 `X-Socket-Visitor-Token`
- `GET /api/socket/customer/:id/files/:messageId`: 访客读取本会话附件，请求头携带 `X-Socket-Visitor-Token`

新会话在服务端按来源 IP 限制为每分钟最多 3 个，独立聊天页还会同步禁用频繁点击。访客发出的第一条文字消息会自动成为会话标题，之后可在咨询页手动修改；刷新页面时会在 10 秒断线宽限内自动恢复原会话，不会被误判为关闭，持续离线超过宽限时间后才关闭且原访客链接不能重新接入。意外断线弹窗可选择继续重连、开启新咨询或结束当前咨询。所有登录页面都会连接全局通知 WebSocket，并在右下角提示访客上线和后台账号登录；刚完成登录的用户本人也会收到登录成功提示。用户、部门、角色、菜单、文章、文件、个人资料及 Socket 等主要业务操作统一使用页面顶部居中的 Message 反馈。

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
$credentials = @{username='your-account'; password='your-password'} | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:8080/api/auth/login -Method Post -ContentType 'application/json' -Body $credentials -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/articles -Method Post -ContentType 'application/json' -Body '{"title":"生产日报","category":"通知公告","author":"管理员","status":"草稿","summary":"今日生产摘要","content":"正文内容"}' -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/files -Method Post -Form @{file=Get-Item .\README.md; displayName='README'; category='文档'; description='项目说明'} -WebSession $session
Invoke-RestMethod -Uri http://localhost:8080/api/auth/logout -Method Post -WebSession $session
```
