#!/bin/bash
# 开发环境启动脚本

set -e

echo "=== 启动开发环境 ==="

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "错误: Docker 未运行，请先启动 Docker"
    exit 1
fi

# 启动依赖服务
echo ">>> 启动 PostgreSQL 和 ZooKeeper..."
docker-compose -f docker-compose.dev.yml up -d

# 等待服务就绪
echo ">>> 等待服务就绪..."
sleep 5

# 检查服务状态
echo ">>> 服务状态:"
docker-compose -f docker-compose.dev.yml ps

echo ""
echo "=== 开发环境已就绪 ==="
echo ""
echo "PostgreSQL: localhost:5432"
echo "  用户: postgres"
echo "  密码: postgres"
echo "  数据库: yaf_config"
echo ""
echo "ZooKeeper: localhost:2181"
echo ""
echo "接下来请分别启动后端和前端:"
echo "  后端: cd backend && go run ./cmd/server"
echo "  前端: cd frontend && npm run dev"

