# Linux 生产部署手册

本文说明如何在一台 Linux 服务器上使用 Docker Compose 部署采集数据平台，并按需安装宿主机终端代理。命令默认在仓库根目录执行，示例部署目录为 `/opt/Agents_BackEnd`。

## 1. 部署结构

| 组件 | 运行位置 | 宿主机示例端口 | 容器端口 | 对外域名示例 |
| --- | --- | --- | --- | --- |
| 后端 API | Docker Compose | `30003` | `8080` | `api.example.com` |
| 管理前端 | Docker Compose | `30004` | `3000` | `admin.example.com` |
| C 端门户 | Docker Compose | `30005` | `3001` | `portal.example.com` |
| Redis | Docker Compose 内部网络 | 不公开 | `6379` | 无 |
| 宿主机代理 | Linux systemd | 主动连接后端 | 无监听端口 | 无 |

生产环境建议只向公网开放 `80`、`443` 和必要的 SSH 管理端口。`30003`、`30004`、`30005` 仅作为 Nginx 本机上游端口，并通过安全组或防火墙禁止公网直接访问。

## 2. 前置条件

服务器需要具备：

- 受支持的 64 位 Linux 系统。
- Git。
- Docker Engine 与 Docker Compose v2。
- Nginx，以及可用的 HTTPS 证书或 Certbot。
- 三个已经解析到服务器公网 IP 的域名。

检查运行环境：

```bash
git --version
docker --version
docker compose version
nginx -v
```

## 3. 获取代码

首次部署：

```bash
cd /opt
git clone https://github.com/lysimportant/Agents_BackEnd.git
cd /opt/Agents_BackEnd
```

已有仓库更新代码：

```bash
cd /opt/Agents_BackEnd
git status --short
git pull --ff-only origin main
```

如果 `git status --short` 显示源码改动，应先确认改动来源，不要使用 `git reset --hard` 覆盖服务器上的未知修改。

## 4. 配置生产环境

首次部署时创建本地环境文件并限制读取权限：

```bash
cd /opt/Agents_BackEnd
cp .env.example .env
chmod 600 .env
mkdir -p config
touch config/email.txt
chmod 600 config/email.txt
```

以下配置使用 `30003`、`30004`、`30005` 作为宿主机端口。将示例域名替换为实际域名：

```dotenv
# 宿主机映射端口
BACKEND_PORT=30003
FRONTEND_PORT=30004
PORTAL_PORT=30005

# 容器内部端口
BACKEND_INTERNAL_PORT=8080
PORTAL_INTERNAL_PORT=3001

# 管理前端公开地址
FRONTEND_API_BASE_URL=https://api.example.com
FRONTEND_SITE_URL=https://admin.example.com

# C 端门户公开地址
PORTAL_API_BASE_URL=https://api.example.com
PORTAL_SITE_URL=https://portal.example.com
PORTAL_ENABLE_CUSTOMER_CHAT=false
PORTAL_REVALIDATE_SECONDS=60

# 浏览器允许来源，必须填写完整 Origin，不带路径和末尾斜杠
CORS_ALLOWED_ORIGINS=https://admin.example.com,https://portal.example.com

# 业务数据持久化目录
DATA_DIR=./backend/data
UPLOAD_DIR=./backend/uploads

# SMTP 配置文件
EMAIL_CONFIG_FILE=./config/email.txt

# 非空 SMTP 环境变量优先于 EMAIL_CONFIG_FILE
EMAIL_HOST=
EMAIL_PORT=
EMAIL_SECURE=
EMAIL_USER=
EMAIL_PASS=
EMAIL_FROM=

# Redis、会话和验证码
REDIS_PASSWORD=
REDIS_DB=0
SESSION_COOKIE_NAME=sessionId
SESSION_TTL_HOURS=8
PASSWORD_CODE_TTL_SECONDS=180
VISITOR_LOG_RETENTION_DAYS=90
COOKIE_SAMESITE=None
COOKIE_SECURE=true

# 不使用部署机直连时保持为空
HOST_AGENT_TOKEN=
```

注意事项：

- `FRONTEND_API_BASE_URL`、`FRONTEND_SITE_URL`、`PORTAL_API_BASE_URL`、`PORTAL_SITE_URL` 和客服开关会进入前端构建产物，修改后必须重新构建镜像。
- 修改 `PORTAL_PORT` 时同步检查 `PORTAL_SITE_URL` 与 `CORS_ALLOWED_ORIGINS`。
- `backend/data/`、SQLite 的 WAL/SHM 文件和 `backend/uploads/` 是业务数据，不得用删除或清空目录的方式解决部署问题。
- `.env`、`config/email.txt` 和宿主机代理令牌不得提交到 Git，也不得输出到公开日志。

如果使用 SMTP 文件，在 `config/email.txt` 中配置：

```dotenv
EMAIL_HOST=smtp.example.com
EMAIL_PORT=465
EMAIL_SECURE=true
EMAIL_USER=mailer@example.com
EMAIL_PASS=replace-with-real-password
EMAIL_FROM=mailer@example.com
```

## 5. 检查并启动 Compose

先展开并检查 Compose 配置：

```bash
cd /opt/Agents_BackEnd
docker compose config --quiet
```

构建并启动 Redis、后端、管理前端和 C 端门户：

```bash
docker compose up -d --build
docker compose ps
```

本机验证：

```bash
curl -fsS http://127.0.0.1:30003/health
curl -fsSI http://127.0.0.1:30004/
curl -fsSI http://127.0.0.1:30005/
```

查看日志：

```bash
docker compose logs --tail=100 backend
docker compose logs --tail=100 frontend
docker compose logs --tail=100 portal
```

持续跟踪某个服务时使用 `-f`，完成后按 `Ctrl+C` 只会退出日志跟踪，不会停止容器。

## 6. 配置 Nginx 与 HTTPS

下面使用三个域名分别代理后端、管理前端和 C 端门户。先在 Nginx `http` 作用域内定义 WebSocket 连接映射和全局请求体策略，例如创建 `/etc/nginx/conf.d/00-platform-global.conf`：

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}

# 0 表示不由 Nginx 限制请求体大小，文件管理可上传任意大小的文件。
client_max_body_size 0;
```

创建 `/etc/nginx/conf.d/agents-backend.conf`：

```nginx
server {
    listen 80;
    server_name api.example.com;

    location / {
        proxy_pass http://127.0.0.1:30003;
        proxy_request_buffering off;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}

server {
    listen 80;
    server_name admin.example.com;

    location / {
        proxy_pass http://127.0.0.1:30004;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name portal.example.com;

    location / {
        proxy_pass http://127.0.0.1:30005;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

检查并重新加载 Nginx：

```bash
nginx -t
systemctl reload nginx
```

使用 Certbot 时可签发并自动配置 HTTPS：

```bash
certbot --nginx \
  -d api.example.com \
  -d admin.example.com \
  -d portal.example.com
```

签发后再次验证：

```bash
nginx -t
systemctl reload nginx
curl -fsS https://api.example.com/health
curl -fsSI https://admin.example.com/
curl -fsSI https://portal.example.com/
```

`client_max_body_size 0` 关闭 Nginx 请求体大小限制，`proxy_request_buffering off` 让大文件上传直接流向后端，避免 Nginx 先完整写入临时文件。文件管理本身不设置应用层单文件大小上限，但实际可上传大小仍受磁盘空间、反向代理上游、网络超时和操作系统资源约束。

后端代理必须保留 WebSocket `Upgrade`，否则 SSH 终端、内部聊天、Socket 客服和部署机直连会连接失败。生产环境不要把 `CORS_ALLOWED_ORIGINS` 设置为通配符。

## 7. 安装宿主机代理（可选）

`collector-host-agent` 是运行在 Linux 宿主机上的 root systemd 进程，不是 Docker Compose 服务。它主动连接后端，为超级管理员提供部署机终端和文件操作能力；不使用该能力时可跳过本节并保持 `HOST_AGENT_TOKEN` 为空。

### 7.1 生成并配置令牌

生成至少 32 字节的随机令牌：

```bash
openssl rand -hex 32
```

把同一个令牌写入仓库 `.env` 的 `HOST_AGENT_TOKEN`，然后只重建后端：

```bash
cd /opt/Agents_BackEnd
docker compose up -d --force-recreate backend
```

不要在终端记录、截图、工单或 Git 提交中公开真实令牌。

### 7.2 构建代理二进制

代理必须在目标 Linux 架构上构建。下面同时支持常见的 AMD64 与 ARM64：

```bash
case "$(uname -m)" in
  x86_64) HOST_AGENT_ARCH=amd64 ;;
  aarch64|arm64) HOST_AGENT_ARCH=arm64 ;;
  *) echo "不支持的架构: $(uname -m)"; exit 1 ;;
esac

docker build \
  --build-arg TARGETARCH="${HOST_AGENT_ARCH}" \
  --file backend/Dockerfile.host-agent \
  --output type=local,dest=./dist \
  backend
```

### 7.3 安装 systemd 服务

以 root 安装二进制：

```bash
install -m 0755 \
  dist/collector-host-agent \
  /usr/local/bin/collector-host-agent

install -m 0644 \
  deploy/host-agent/collector-host-agent.service \
  /etc/systemd/system/collector-host-agent.service
```

首次安装时创建环境文件：

```bash
install -m 0600 \
  deploy/host-agent/collector-host-agent.env.example \
  /etc/collector-host-agent.env
```

编辑 `/etc/collector-host-agent.env`。代理与后端位于同一台宿主机时，优先使用回环地址绕过公网和 Nginx：

```dotenv
HOST_AGENT_SERVER_URL=ws://127.0.0.1:30003/api/server/host-agent
HOST_AGENT_TOKEN=replace-with-the-same-token-from-project-env
HOST_AGENT_NAME=production-server
HOST_AGENT_SHELL=/bin/bash
HOST_AGENT_RECONNECT_SECONDS=5
```

限制权限并启动：

```bash
chown root:root /etc/collector-host-agent.env
chmod 600 /etc/collector-host-agent.env
systemctl daemon-reload
systemctl enable --now collector-host-agent
systemctl status collector-host-agent --no-pager
```

查看日志：

```bash
journalctl -u collector-host-agent -n 100 --no-pager
```

服务显示 `active (running)` 且日志报告连接成功后，以超级管理员登录管理前端验证“部署机直连”。系统管理员和普通用户不能使用该入口。

确认代理实际以 root 运行：

```bash
ps -o user,group,pid,args -C collector-host-agent
```

输出中的 `USER` 和 `GROUP` 应为 `root`。root 运行会让部署机直连具备整台主机的读写权限，只应在可信的超级管理员环境启用；仍然不要给后端容器添加 `privileged`、Docker Socket 或宿主机根目录挂载。

### 7.4 部署机直连与 SSH 端口

管理前端的“部署机直连”使用宿主机上的 `collector-host-agent` 主动建立 WebSocket，不是浏览器直接 SSH 到服务器，因此不需要开放或使用 `22` 端口。代理与后端在同一台宿主机时，优先使用：

```dotenv
HOST_AGENT_SERVER_URL=ws://127.0.0.1:30003/api/server/host-agent
```

如果代理通过 Nginx 访问后端，则使用后端域名的 `wss://` 地址，并确保 Nginx 转发 WebSocket。只有管理前端的普通“SSH”连接模式才需要目标主机的 SSH 端口（默认 `22`，也可以是自定义端口）；该连接由后端访问目标主机，浏览器不直接访问目标主机的 `22` 端口。

## 8. 日常更新

更新前先确认容器状态和工作区：

```bash
cd /opt/Agents_BackEnd
docker compose ps
git status --short
git log -1 --oneline
```

拉取最新代码并重建：

```bash
git pull --ff-only origin main
docker compose config --quiet
docker compose up -d --build
docker compose ps
```

如果代码已经通过其他方式更新到服务器，并且当前目录就是 `/opt/Agents_BackEnd`，可以直接执行：

```bash
docker compose up -d --build
```

`git pull --ff-only origin main` 只负责更新源码，`docker compose config --quiet` 只负责提前检查配置；两者都不是镜像构建的必需步骤。`docker compose up -d --build` 会使用当前目录源码重建需要更新的镜像并后台启动 Compose 服务。不要默认添加 `--remove-orphans`，它可能删除同一 Compose 项目中但不在当前文件声明的其他容器。

### 8.1 防止 B 端终端断开导致构建中断

`-d` 只会让容器在启动后脱离终端，镜像构建过程仍然运行在当前 shell 中。如果后端容器重启会使 B 端 WebSocket 断开，建议把部署任务交给宿主机 `systemd`：

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

正常提交后终端只会显示 `Running as unit: ...service` 并返回提示符，不应继续显示 BuildKit 构建输出。命令提交后即可关闭 B 端终端；构建和容器重建由宿主机服务继续执行。动态单元名可以避免上一次部署仍在运行时发生名称冲突。重新连接后使用实际输出的单元名称检查结果：

```bash
journalctl -u agents-deploy-实际时间戳.service -n 200 --no-pager
cd /opt/Agents_BackEnd
docker compose ps
```

`--remain-after-exit` 会保留任务状态，便于查看 `systemctl status` 和 `journalctl`。该方案可以抵抗 B 端、后端或 Compose 容器重启；如果整台 Linux 主机发生真正的系统重启，正在执行的构建仍会中断，需要另行配置持久化部署服务或 CI/CD 重试。

如果本次更新包含 `backend/cmd/host-agent/`、`backend/Dockerfile.host-agent` 或 `deploy/host-agent/`，重新构建并安装代理二进制，然后执行：

```bash
systemctl daemon-reload
systemctl restart collector-host-agent
systemctl status collector-host-agent --no-pager
```

更新后重复执行后端健康检查、三个站点检查和关键 WebSocket 功能验收。

## 9. 备份与数据保护

需要制作一致性文件备份时，先短暂停止后端写入，再备份数据库、上传目录和部署配置：

```bash
cd /opt/Agents_BackEnd
BACKUP_TIMESTAMP=$(date +%Y%m%d-%H%M%S)
mkdir -p "/opt/Agents_BackEnd-backups/${BACKUP_TIMESTAMP}"
docker compose stop backend
tar -czf "/opt/Agents_BackEnd-backups/${BACKUP_TIMESTAMP}/business-data.tar.gz" \
  backend/data \
  backend/uploads
docker compose start backend
```

另行安全备份 `.env`、SMTP 配置和 `/etc/collector-host-agent.env`，这些文件包含秘密，不应与普通源码归档混放或上传到公共位置。

禁止执行以下操作：

- 不要删除 `backend/data/`、SQLite、WAL 或 SHM 文件来处理迁移或启动错误。
- 不要清空 `backend/uploads/`。
- 不要执行 `docker compose down -v`，它会删除 Redis 持久卷。
- 不要通过真实业务 API 批量删除所谓“验收数据”。

## 10. 故障排查

### Compose 服务未启动

```bash
docker compose config --quiet
docker compose ps -a
docker compose logs --tail=200 backend
docker compose logs --tail=200 frontend
docker compose logs --tail=200 portal
```

后端未通过健康检查时，先检查 SQLite/上传目录权限、Redis 状态、邮件配置挂载和环境变量，不要删除数据库。

### 前端仍使用旧域名或旧端口

`NEXT_PUBLIC_*` 配置在构建时进入客户端产物。修改 `.env` 后执行：

```bash
docker compose up -d --build frontend portal
```

### 浏览器出现 CORS 错误

确认 `CORS_ALLOWED_ORIGINS` 包含浏览器地址栏中的完整 Origin，例如 `https://admin.example.com` 和 `https://portal.example.com`。不要填写路径，也不要只填写容器服务名。

### WebSocket 无法连接

检查 Nginx 是否传递 `Upgrade`、`Connection`，并确认读取超时足够长：

```bash
nginx -t
journalctl -u nginx -n 100 --no-pager
docker compose logs --tail=200 backend
```

### `collector-host-agent.service` 不存在

环境文件不等于 systemd 服务。安装仓库提供的 unit 后重新加载：

```bash
install -m 0644 \
  deploy/host-agent/collector-host-agent.service \
  /etc/systemd/system/collector-host-agent.service
systemctl daemon-reload
systemctl enable --now collector-host-agent
```

### 宿主机代理反复重连

```bash
systemctl status collector-host-agent --no-pager
journalctl -u collector-host-agent -n 200 --no-pager
curl -fsS http://127.0.0.1:30003/health
```

确认项目 `.env` 和 `/etc/collector-host-agent.env` 中的令牌完全一致，并在修改项目 `.env` 后重建后端容器。日志中不要打印令牌。

## 11. 上线验收清单

- `docker compose ps` 中 Redis 为 `healthy`，后端为 `healthy`，两个前端均为运行状态。
- `https://api.example.com/health` 返回成功。
- 管理前端可以登录、刷新页面并访问受权限保护的功能。
- C 端门户可以加载公开文章、图片与资源。
- 管理前端和门户的浏览器控制台没有 CORS 或混合内容错误。
- SSH、内部聊天、Socket 客服等 WebSocket 功能按实际启用范围验证通过。
- 启用宿主机代理时，systemd 服务为 `active (running)`，超级管理员可以连接，其他角色被拒绝。
- 防火墙或安全组未向公网开放 Redis 和内部上游端口。
- `.env`、SMTP 密码、代理令牌、SQLite 和上传内容均未进入 Git。
- 已完成业务数据备份，并验证备份文件位于受控位置。
