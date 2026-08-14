# Docker Compose 启动说明

根目录的 `docker-compose.yml` 会同时启动后端、前端和 Redis。

```powershell
Copy-Item .env.example .env
docker compose up -d --build
```

默认地址：

- 前端：`http://localhost:3000`
- 后端健康检查：`http://localhost:8080/health`

端口和宿主机持久化目录集中在根目录 `.env`：

```dotenv
BACKEND_PORT=8080
FRONTEND_PORT=3000
FRONTEND_API_BASE_URL=http://localhost:8080
FRONTEND_SITE_URL=http://localhost:3000
DATA_DIR=./data
UPLOAD_DIR=./uploads
CONFIG_DIR=./config
```

修改端口后，`FRONTEND_API_BASE_URL` 和 `FRONTEND_SITE_URL` 也要同步修改。SQLite 数据保存在 `DATA_DIR`，上传文件保存在 `UPLOAD_DIR`，Redis 数据由 Compose 卷 `redis-data` 持久化。SMTP 配置可选，放入 `CONFIG_DIR/email.txt`。
