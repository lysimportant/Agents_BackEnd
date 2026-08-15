# Docker Compose 故障排查指南

## 最新修复（2026-08-15）

### 问题：Backend 容器 unhealthy - /health 返回 404

**根因**：Docker healthcheck 使用 `wget --spider` 发送 HEAD 请求，但后端路由只注册了 GET 方法。

**修复**：在 `backend/routes/routes.go` 中同时注册 GET 和 HEAD 方法：

```go
healthHandler := func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
router.GET("/health", healthHandler)
router.HEAD("/health", healthHandler)
```

**如何应用修复**：

```bash
cd /opt/Agents_BackEnd

# 停止容器
docker compose down

# 重新构建后端（代码已修改）
docker compose build backend

# 启动所有服务
docker compose up -d

# 查看状态（等待约 10-15 秒）
docker compose ps

# 验证健康检查
curl -I http://localhost:30003/health
```

## 当前修复内容

已修复自定义端口配置问题，现在支持通过 `.env` 文件完全控制端口：

1. **backend/Dockerfile**：接收 `BACKEND_INTERNAL_PORT` 构建参数
2. **docker-compose.yml**：
   - backend 服务的 ports、environment、healthcheck 全部使用环境变量
   - frontend 服务的构建参数 `BACKEND_INTERNAL_URL` 使用环境变量

## 如何排查 backend 启动失败

### 步骤 1：检查 .env 文件

确保 `.env` 文件存在且格式正确：

```bash
cat .env
```

应该看到：
```env
BACKEND_PORT=30003
FRONTEND_PORT=30004
BACKEND_INTERNAL_PORT=8080
```

### 步骤 2：停止所有容器

```bash
docker compose down
```

### 步骤 3：查看详细启动日志

```bash
docker compose up --build
```

**不要使用 `-d` 参数**，这样可以直接看到所有容器的实时日志。

### 步骤 4：分析常见错误

#### 错误 A：Redis 连接失败
```
连接 Redis 失败: dial tcp ...
```
**原因**：Redis 容器未启动或健康检查未通过  
**解决**：等待 Redis 容器完全启动（约 10-15 秒）

#### 错误 B：SQLite 路径权限问题
```
打开 SQLite 数据库失败: unable to open database file
```
**原因**：`/app/data` 目录不存在或无写权限  
**解决**：检查 `DATA_DIR` 映射的宿主机目录是否存在且有权限

#### 错误 C：端口已被占用
```
bind: address already in use
```
**原因**：宿主机端口已被其他程序占用  
**解决**：
```bash
# Windows
netstat -ano | findstr :30003
taskkill /PID <PID> /F

# Linux/Mac
lsof -i :30003
kill -9 <PID>
```

#### 错误 D：健康检查超时
```
Container backend is unhealthy
```
**原因**：backend 服务启动慢或健康检查配置错误  
**解决**：查看 backend 日志，检查是否有 Go 运行时错误

### 步骤 5：单独测试 backend

如果 frontend 依赖导致问题复杂化，可以单独启动 backend：

```bash
docker compose up redis backend
```

### 步骤 6：进入容器内部调试

如果容器已经运行但健康检查失败：

```bash
# 查看容器状态
docker compose ps

# 进入 backend 容器
docker compose exec backend sh

# 在容器内测试
wget --spider --quiet http://127.0.0.1:8080/health && echo "OK" || echo "Failed"

# 查看环境变量
env | grep -E "SERVER_ADDRESS|BACKEND|REDIS"

# 测试 Redis 连接
ping redis
```

## 完整重置步骤

如果问题持续存在，执行完整重置：

```bash
# 1. 停止并删除所有容器、网络、卷
docker compose down -v

# 2. 删除构建缓存
docker builder prune -f

# 3. 确保 .env 文件正确
cat .env

# 4. 重新构建和启动
docker compose up --build
```

## 验证配置

修复后，验证端口配置是否生效：

```bash
# 1. 检查 backend 容器端口映射
docker compose ps backend

# 应该显示：
# 0.0.0.0:30003->8080/tcp

# 2. 测试后端健康检查
curl http://localhost:30003/health

# 3. 测试前端访问
curl http://localhost:30004
```
