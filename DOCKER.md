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
BACKEND_INTERNAL_PORT=8080
FRONTEND_API_BASE_URL=http://localhost:8080
FRONTEND_SITE_URL=http://localhost:3000
DATA_DIR=./backend/data
UPLOAD_DIR=./backend/uploads
CONFIG_DIR=./config
```

**端口配置说明：**
- `BACKEND_PORT`：宿主机映射端口（浏览器访问的端口）
- `BACKEND_INTERNAL_PORT`：容器内部端口（通常不需要修改）
- `FRONTEND_PORT`：前端宿主机端口

修改 `BACKEND_PORT` 后，`FRONTEND_API_BASE_URL` 也要同步修改为新端口。

SQLite 数据保存在 `DATA_DIR`，上传文件保存在 `UPLOAD_DIR`，Redis 数据由 Compose 卷 `redis-data` 持久化。SMTP 配置可选，放入 `CONFIG_DIR/email.txt`。
