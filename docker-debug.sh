#!/bin/bash

echo "=== Docker Compose 诊断脚本 ==="
echo ""

echo "1. 检查 .env 文件内容："
cat .env | grep -E "BACKEND_PORT|BACKEND_INTERNAL_PORT|FRONTEND"
echo ""

echo "2. 查看后端容器日志（最后 50 行）："
docker compose logs backend --tail=50
echo ""

echo "3. 检查后端容器状态："
docker compose ps backend
echo ""

echo "4. 尝试进入后端容器并测试健康检查："
docker compose exec backend wget --spider --quiet http://127.0.0.1:8080/health && echo "健康检查成功" || echo "健康检查失败"
echo ""

echo "5. 检查后端容器环境变量："
docker compose exec backend env | grep -E "SERVER_ADDRESS|BACKEND"
echo ""

echo "6. 检查 Redis 连接："
docker compose exec backend wget --spider --quiet http://redis:6379 && echo "Redis 可达" || echo "Redis 不可达"
echo ""

echo "=== 诊断完成 ==="
