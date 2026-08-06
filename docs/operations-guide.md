# 平台操作说明

这份说明面向第一次运行平台的开发人员和管理员。按照“先后端、再前端、最后浏览器操作”的顺序执行即可。

如果要开发新的管理页面和对应 API，请阅读[前后端页面/API 开发说明](./frontend-backend-page-api-guide.md)。

## 一、准备环境

请先安装：

- Go 1.26 或兼容版本
- Node.js 和 npm
- PowerShell

项目分为两个独立服务：

- 后端 API：`http://localhost:8080`
- 前端管理界面：`http://localhost:3000`

## 二、启动后端

打开第一个 PowerShell 窗口：

```powershell
cd D:\agent\backend
go mod download
go run .
```

看到服务监听 `:8080` 后，打开另一个 PowerShell 检查后端：

```powershell
Invoke-RestMethod http://localhost:8080/health
```

能返回健康状态，就说明后端已经启动。

### 后端数据位置

默认情况下，业务数据和上传文件保存在：

- SQLite 数据库：`backend/data/app.db`
- 普通文件：`backend/uploads/`
- 客服聊天附件：`backend/uploads/socket/`
- 内部聊天附件：`backend/uploads/internal-chat/`

首次启动会自动创建表、索引、菜单、角色和默认管理员，不会覆盖已有业务数据。

### 使用隔离数据测试

测试上传、聊天附件或数据库迁移时，请使用独立目录，不要碰正式数据：

```powershell
cd D:\agent\backend
$env:SQLITE_PATH="D:\agent\.workspace-temp\p30-operations\app.db"
$env:UPLOAD_DIR="D:\agent\.workspace-temp\p30-operations\uploads"
go run .
```

## 三、启动前端

打开第二个 PowerShell 窗口：

```powershell
cd D:\agent\frontend
npm install
$env:NEXT_PUBLIC_API_BASE_URL="http://localhost:8080"
npm run dev
```

浏览器打开 <http://localhost:3000>。

开发时保持后端窗口和前端窗口都运行。修改代码后，前端页面通常会自动刷新，后端 Go 代码需要重新启动服务。

## 四、第一次登录

默认管理员账号为：

- 用户名：`MH`
- 初始密码：`123`

登录后建议立即打开个人资料修改密码和联系方式。密码不会显示在页面或 API 返回中。

## 五、常用后台操作

### 1. 用户、部门、角色和菜单

在工作台左侧进入对应菜单：

1. 先创建部门。
2. 再创建角色并分配菜单、动作权限。
3. 最后创建用户，选择部门和角色。
4. 用户登录后只能看到自己有效菜单，按钮权限由后端再次校验。

超级管理员和系统管理员拥有全部当前权限。只有超级管理员可以创建或调整超级管理员及系统管理员。

### 2. 文件管理

进入“文件管理”后可以：

- 点击刷新按钮重新获取文件列表。
- 预览或下载普通文件和图片。
- 将图片设置为登录背景。
- 删除文件时先进入回收站，需要时可以恢复。
- 超级管理员可以看到聊天附件分类“聊天数据”，并查看内部聊天和客服聊天的附件。

聊天附件仍然受权限保护，不能通过猜测静态地址访问。

### 3. 访问分析

进入“访问分析”可以查看访问者的 IP、User-Agent、来源页、国家/地区等元数据，并使用：

- 时间筛选和关键词搜索
- 每页 10、20、30、50 或 100 条
- 刷新按钮
- 表格、统计卡片和图表

访问日志默认保留 90 天，仅超级管理员和系统管理员可查看。国家/地区需要可信反向代理写入 `CF-IPCountry` 等请求头。

## 六、内部聊天 `/chat`

登录后可从页面右上角聊天入口，或直接打开 <http://localhost:3000/chat>。

1. 左侧选择群聊或在线用户发起私聊。
2. 在输入框输入文字，按 Enter 发送；需要换行时使用 Shift+Enter。
3. 点击表情按钮，在分类面板中选择表情，表情会插入当前文字位置。
4. 点击附件按钮选择图片或普通文件，等待上传完成后再发送消息。
5. 图片附件会显示缩略图，普通文件显示名称、大小和下载入口。
6. 消息正文中的 HTTP/HTTPS 图片链接会显示预览，其他网址会保留为安全外链。
7. 左侧用户行和聊天入口会显示未读数量；收到新消息时会有视觉提示和可用的声音提示。

内部聊天和客服聊天是两套独立业务，内部聊天的接口统一位于 `/api/internal-chat/*`。

## 七、客服聊天

客服功能位于工作台的“Socket 客服”页面，面向网站访客；它不是内部聊天 `/chat`。

管理员可以在客服页面查看访客会话、发送文字/表情/图片/文件、结束会话，并在文件管理的“聊天数据”分类中查看聊天附件。网站访客使用客服悬浮组件发起咨询，不需要后台账号。

## 八、生产构建和停止服务

前端发布前执行：

```powershell
cd D:\agent\frontend
npx.cmd tsc --noEmit --incremental false
npm run build
npm run start
```

停止开发服务时，在对应 PowerShell 窗口按 `Ctrl+C`。不要手工删除 `backend/data/`、`backend/uploads/` 或其 SQLite WAL/SHM 文件。

## 九、常见问题

### 页面打不开

确认后端的 `http://localhost:8080/health` 正常，再确认前端运行在 `3000` 端口。端口被占用时，先关闭旧进程或改用其他端口。

### 登录后 API 返回 401/403

确认前后端地址、Cookie 和 CORS 配置一致。跨站 HTTPS 部署通常需要：

```powershell
$env:COOKIE_SAMESITE="None"
$env:COOKIE_SECURE="true"
$env:CORS_ALLOWED_ORIGINS="https://你的前端域名"
```

### 附件上传失败

确认 `UPLOAD_DIR` 可写、文件没有超过接口限制，并检查后端窗口的错误日志。验收附件功能时请使用隔离的 `SQLITE_PATH` 和 `UPLOAD_DIR`。
