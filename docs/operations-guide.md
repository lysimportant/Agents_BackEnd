# 平台操作说明

这份说明面向第一次运行平台的开发人员和管理员。系统由后端、B 端管理前端和 C 端内容门户三个独立应用组成。

## 一、准备环境

请安装 Go 1.26、Node.js、npm 和 PowerShell。默认地址：

| 服务 | 地址 | 作用 |
| --- | --- | --- |
| 后端 | `http://localhost:8080` | API、SQLite、Redis 会话、文件、聊天和终端 |
| B 端 | `http://localhost:3000` | 管理后台、内部聊天、客服管理和 SSH 工作区 |
| C 端 | `http://localhost:3001` | 公开文章、图片、资源、分类和搜索 |

## 二、启动后端

```powershell
cd D:\agent\backend
go mod download
go run .
```

健康检查：

```powershell
Invoke-RestMethod http://localhost:8080/health
```

默认业务数据位置：

- SQLite：`backend/data/app.db`
- 文件管理上传：`backend/uploads/`
- 客服附件：`backend/uploads/socket/`
- 内部聊天附件：`backend/uploads/internal-chat/`

首次启动会增量创建表、索引、菜单、角色和默认管理员，不会覆盖已有业务数据。测试迁移、上传或聊天时必须使用隔离目录：

```powershell
cd D:\agent\backend
$env:SQLITE_PATH="D:\agent\.workspace-temp\<task>\app.db"
$env:UPLOAD_DIR="D:\agent\.workspace-temp\<task>\uploads"
go run .
```

## 三、启动 B 端

```powershell
cd D:\agent\frontend
npm install
$env:NEXT_PUBLIC_API_BASE_URL="http://localhost:8080"
npm run dev
```

浏览器打开 `http://localhost:3000`。管理页主要在根路由 `/` 内通过 `activePage` 切换；内部聊天使用真实路由 `/chat`，Socket 客服访客页使用 `/socket/chat/...`。

## 四、启动 C 端

```powershell
cd D:\agent\portal
npm install
$env:NEXT_PUBLIC_API_BASE_URL="http://localhost:8080"
$env:NEXT_PUBLIC_SITE_URL="http://localhost:3001"
npm run dev
```

浏览器打开 `http://localhost:3001`，根地址会按语言偏好跳转，`/{locale}` 当前继续跳转到 `/{locale}/images`。支持 `zh-CN`、`en-US`、`ja-JP`。

C 端图片页没有分页按钮。页面内部仍按后端分页批次读取数据，默认每批 24 条、最多 50 条，接近底部时自动预加载下一批，并显示加载或重试状态。缩略图用于低流量占位，中图用于瀑布流，打开预览后才读取当前原图；已加载卡片不会因离开视野而卸载。

## 五、第一次登录

初始化超级管理员：

- 用户名：`MH`
- 初始密码：`123`

首次登录后应立即修改密码和联系方式。`MH` 只在用户表首次为空时创建，后续不具有特殊业务权限。

C 端也可以复用后端登录会话。匿名访问不会看到 18R 内容；登录后开启“显示 18R 内容”才会写入 `portal-r18=1` Cookie，并让公开接口包含 18R 内容。会话 ID 始终保存在 HttpOnly Cookie 中。

## 六、常用后台操作

### 用户、部门、角色和菜单

1. 创建部门。
2. 创建角色并分配菜单。
3. 创建用户并选择部门和角色。
4. 在用户页按需补充个人菜单和动作权限。

超级管理员和系统管理员拥有全部菜单与动作。只有超级管理员可以管理其他超级管理员。

### 文件管理

文件管理支持：

- 一次选择并上传多个文件。
- 前端过滤同一批选择中的重复文件。
- 后端按“同一所有者 + SHA-256 内容哈希”过滤有效重复文件，后端结果是权威判断。
- 文件管理上传不设置应用层单文件大小上限，前端上传请求也不主动设置超时。
- 标签会去首尾空白、移除前导 `#` 并按不区分大小写去重，最多 12 个、每个最多 24 个 Unicode 字符；同一文件可在元数据编辑时更新标签。
- 预览、下载、刷新、编辑元数据和将图片设为登录背景。
- 默认软删除到回收站，确认后才可永久删除物理文件。
- 超级管理员查看内部聊天和客服聊天附件分类。

文件管理图片缩略图使用受保护的 `/api/files/:id/thumbnail`。`.thumbnail-cache` 仅是可再生成的派生缓存，可能占用磁盘但不属于业务文件；缓存缺失时会重新生成，不要手工清理正式上传目录。

C 端图片预览支持复制原图 URL、下载原图、点赞切换、标签和评论。互动读取可匿名，点赞与评论要求登录；评论去首尾空白后最多 500 个 Unicode 字符，最新列表最多返回 100 条。

聊天和文章附件仍保留各自的数量、大小或类型限制，不能把文件管理的无限制规则套用到其他接口。

### 文章和 C 端公开规则

C 端文章公开条件是非私密且状态为“已发布”。C 端文件公开条件是非私密且未进入回收站。当前没有额外的门户发布开关；调整私密状态、文章状态或文件回收站状态会直接影响 C 端可见性。

### 访问分析

访问分析支持时间和关键词筛选、刷新、统计卡片、图表和每页 `10/20/30/50/100` 条。日志默认保留 90 天；国家/地区需要可信反向代理提供 `CF-IPCountry` 等 Header。

## 七、聊天和终端

内部聊天 `/chat` 与 Socket 客服是两套独立系统，状态、WebSocket 和附件权限不得混用。历史消息浏览时，只有用户接近底部或主动查看最新消息才自动跟随新消息。

Header 中的 SSH 工作区对所有有效登录用户开放。只有 `roleCode=super-admin` 可以选择“部署机直连”；该入口连接 Linux 宿主机上的独立 root systemd 代理，因此具备 root 文件和命令权限。

### Compose 更新不中断

如果直接在 B 端终端执行 `docker compose up -d --build`，后端容器重建时 WebSocket/PTY 可能断开。生产更新使用宿主机动态 systemd 单元：

```bash
deploy_unit="agents-deploy-$(date +%Y%m%d-%H%M%S)"

systemd-run \
  --unit="$deploy_unit" \
  --description="Agents_BackEnd deployment" \
  --working-directory=/opt/Agents_BackEnd \
  --property=Type=oneshot \
  --property=TimeoutStartSec=infinity \
  --remain-after-exit \
  --no-block \
  /bin/bash -lc '
    set -Eeuo pipefail
    git pull --ff-only origin main
    docker compose config --quiet
    docker compose up -d --build
    docker compose ps
  '

echo "部署单元：$deploy_unit"
```

终端断开后查看结果：

```bash
systemctl status "$deploy_unit" --no-pager
journalctl -u "$deploy_unit" -n 100 --no-pager
docker compose ps
```

看到 `Finished ...service` 且 backend healthy、frontend/portal started、redis healthy，才算更新完成。该部署机直连模式不需要 SSH/22 端口；普通 SSH 工作区仍由后端连接目标主机 SSH 端口。

## 八、验证和生产构建

后端：

```powershell
cd D:\agent\backend
go test ./...
go vet ./...
```

B 端：

```powershell
cd D:\agent\frontend
.\node_modules\.bin\tsc.cmd --noEmit --incremental false
npm run build
```

C 端：

```powershell
cd D:\agent\portal
npm run typecheck
npm run lint
npm run docs
npm run build
```

前端开发服务和生产构建共享 `.next`，不要并发运行。停止服务时在对应终端按 `Ctrl+C`，不要删除业务 SQLite、WAL/SHM 或上传目录。

## 九、常见问题

### 页面打不开

先检查 `http://localhost:8080/health`，再确认 B 端 `3000` 和 C 端 `3001` 端口。端口被占用时关闭旧进程或修改启动端口和对应公开地址。

### 登录后返回 401/403

检查 Cookie、CORS、账号状态、菜单和动作权限。跨站 HTTPS 通常需要：

```powershell
$env:COOKIE_SAMESITE="None"
$env:COOKIE_SECURE="true"
$env:CORS_ALLOWED_ORIGINS="https://管理域名,https://门户域名"
```

### 文件上传失败

文件管理没有应用层大小上限。检查 `UPLOAD_DIR` 是否可写、磁盘空间、Nginx 是否设置 `client_max_body_size 0`、上游代理限制和网络连接。聊天或文章附件失败时，还要检查对应接口自身限制。

### C 端没有显示 18R 内容

需要先在 C 端登录有效账号，再开启 18R 开关。只有 `portal-r18=1` 而没有有效会话时，后端仍会排除 18R 内容。

### 图片滚到底部没有继续加载

检查浏览器控制台和 `/api/public/images?page=...&pageSize=...` 请求。接口仍有每页最大 50 条的批次限制，但页面不会显示分页按钮。
