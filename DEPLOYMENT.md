# YAF 配置中心部署指南

## 部署方式

### 方式一：开发环境（仅依赖服务）

只启动 PostgreSQL 和 ZooKeeper，后端和前端在本地运行：

```bash
# 启动依赖服务
docker-compose -f docker-compose.dev.yml up -d

# 查看服务状态
docker-compose -f docker-compose.dev.yml ps

# 停止服务
docker-compose -f docker-compose.dev.yml down
```

### 方式二：完整部署（生产环境）

使用 Docker Compose 部署所有服务（包括 YAF 容器）：

```bash
# 构建并启动所有服务
docker-compose up -d --build

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down

# 停止并删除数据卷（谨慎使用）
docker-compose down -v
```

## 服务说明

### 1. PostgreSQL 数据库
- **端口**: 5432
- **数据库名**: yaf_config
- **用户名**: postgres
- **密码**: postgres
- **数据卷**: postgres_data

### 2. ZooKeeper
- **端口**: 2181
- **数据卷**: zk_data, zk_datalog

### 3. 后端服务
- **端口**: 8080
- **容器名**: yaf-config-backend
- **API 地址**: http://localhost:8080/api/v1

### 4. 前端服务
- **端口**: 3000
- **容器名**: yaf-config-frontend
- **访问地址**: http://localhost:3000

### 5. YAF 容器
- **容器名**: yaf-production (生产) / yaf-dev (开发)
- **super_mediator 端口**: 18000
- **功能**: 
  - 网络流量捕获（YAF）
  - 流量处理（super_mediator）
  - 配置管理（config-agent）
  - 数据上报（processor）

## YAF 容器网络配置

YAF 容器需要访问网络接口来捕获流量，有两种配置方式：

### 方式一：Host 网络模式（推荐用于生产环境）

在 `docker-compose.yml` 中取消注释：

```yaml
yaf:
  network_mode: host
  # 移除 ports 映射（使用 host 模式时不需要）
```

**优点**：
- 可以直接访问主机网络接口
- 性能更好
- 配置简单

**缺点**：
- 容器与主机共享网络栈
- 端口冲突风险

### 方式二：Bridge 模式 + Privileged（用于开发测试）

在 `docker-compose.yml` 中取消注释：

```yaml
yaf:
  privileged: true
  cap_add:
    - NET_ADMIN
    - NET_RAW
```

**优点**：
- 网络隔离
- 适合开发测试

**缺点**：
- 需要特殊权限
- 可能无法捕获所有流量

## 环境变量配置

### YAF 容器环境变量

可以通过环境变量或 `.env` 文件配置：

```bash
# .env 文件示例
YAF_CLUSTER=production
YAF_NODE_ID=node-1
YAF_INTERFACE=eth0
```

或在 `docker-compose.yml` 中直接设置：

```yaml
environment:
  YAF_CLUSTER: production
  YAF_NODE_ID: node-1
  YAF_INTERFACE: eth0
```

### 后端服务环境变量

```bash
SERVER_PORT=8080
SERVER_MODE=release
DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_DBNAME=yaf_config
ZOOKEEPER_SERVERS=zookeeper:2181
```

## 部署步骤

### 1. 准备环境

确保已安装：
- Docker (20.10+)
- Docker Compose (2.0+)

### 2. 克隆项目

```bash
git clone <repository-url>
cd yf-web
```

### 3. 配置环境变量（可选）

创建 `.env` 文件：

```bash
# 数据库配置
POSTGRES_PASSWORD=your_secure_password

# YAF 配置
YAF_CLUSTER=production
YAF_NODE_ID=node-1
YAF_INTERFACE=eth0
```

### 4. 启动服务

**开发环境**：
```bash
docker-compose -f docker-compose.dev.yml up -d
```

**生产环境**：
```bash
docker-compose up -d --build
```

### 5. 验证部署

```bash
# 检查所有容器状态
docker-compose ps

# 检查后端健康状态
curl http://localhost:8080/health

# 检查 ZooKeeper 连接
docker exec yaf-zookeeper zkServer.sh status

# 查看 YAF 容器日志
docker logs yaf-production -f
```

### 6. 访问服务

- **前端界面**: http://localhost:3000
- **后端 API**: http://localhost:8080/api/v1
- **健康检查**: http://localhost:8080/health

## 数据持久化

所有数据都存储在 Docker volumes 中：

```bash
# 查看 volumes
docker volume ls

# 备份数据
docker run --rm -v yf-web_postgres_data:/data -v $(pwd):/backup \
  alpine tar czf /backup/postgres_backup.tar.gz /data

# 恢复数据
docker run --rm -v yf-web_postgres_data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/postgres_backup.tar.gz -C /
```

## 故障排查

### 1. 端口冲突

如果端口被占用，修改 `docker-compose.yml` 中的端口映射：

```yaml
ports:
  - "8081:8080"  # 改为其他端口
```

### 2. YAF 容器无法捕获流量

- 检查网络模式配置
- 确认有 NET_ADMIN 和 NET_RAW 权限
- 检查网络接口名称是否正确

### 3. ZooKeeper 连接失败

```bash
# 检查 ZooKeeper 状态
docker exec yaf-zookeeper zkServer.sh status

# 查看日志
docker logs yaf-zookeeper
```

### 4. 数据库连接失败

```bash
# 检查数据库状态
docker exec yaf-postgres pg_isready -U postgres

# 查看数据库日志
docker logs yaf-postgres
```

## 更新部署

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker-compose up -d --build

# 或者只更新特定服务
docker-compose up -d --build backend
```

## 清理

```bash
# 停止并删除容器
docker-compose down

# 停止并删除容器和数据卷（谨慎使用）
docker-compose down -v

# 清理未使用的镜像和卷
docker system prune -a
```

## 多节点部署

如果需要部署多个 YAF 节点：

1. 为每个节点创建独立的 docker-compose 文件
2. 设置不同的 `YAF_NODE_ID` 和端口
3. 使用相同的 ZooKeeper 集群

示例：

```yaml
# docker-compose.node1.yml
yaf:
  container_name: yaf-node-1
  environment:
    YAF_NODE_ID: node-1
    SM_LISTEN_PORT: 18000
  ports:
    - "18000:18000"
```

```yaml
# docker-compose.node2.yml
yaf:
  container_name: yaf-node-2
  environment:
    YAF_NODE_ID: node-2
    SM_LISTEN_PORT: 18001
  ports:
    - "18001:18000"
```

## 安全建议

1. **修改默认密码**：生产环境务必修改数据库密码
2. **使用环境变量**：敏感信息通过环境变量或 secrets 管理
3. **网络隔离**：使用 Docker 网络隔离服务
4. **定期备份**：定期备份数据库和配置数据
5. **监控日志**：设置日志监控和告警

## 性能优化

1. **资源限制**：为容器设置 CPU 和内存限制
2. **数据卷优化**：使用本地 SSD 存储数据卷
3. **网络优化**：YAF 容器使用 host 网络模式
4. **日志轮转**：配置日志轮转避免磁盘满
