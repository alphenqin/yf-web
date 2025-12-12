#!/bin/bash
# 构建脚本

set -e

echo "=== 构建 YAF 配置中心 ==="

# 构建后端
echo ">>> 构建后端服务..."
cd backend
go build -ldflags="-w -s" -o ../dist/yaf-config-service ./cmd/server
cd ..
echo ">>> 后端构建完成"

# 构建 Config Agent
echo ">>> 构建 Config Agent..."
cd config-agent
go build -ldflags="-w -s" -o ../dist/yaf-config-agent ./cmd/agent
cd ..
echo ">>> Config Agent 构建完成"

# 构建前端
echo ">>> 构建前端..."
cd frontend
npm install
npm run build
cp -r dist ../dist/frontend
cd ..
echo ">>> 前端构建完成"

echo "=== 构建完成 ==="
echo "输出目录: dist/"
ls -la dist/

