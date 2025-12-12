# YAF 分布式配置中心

基于 ZooKeeper 的 YAF (Yet Another Flowmeter) 分布式配置管理系统，支持全局、集群、节点三级配置管理和动态下发。

## 系统架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Web UI (Vue)  │────▶│  Config Service │────▶│   PostgreSQL    │
│                 │     │     (Go/Gin)    │     │   (配置持久化)   │
└─────────────────┘     └────────┬────────┘     └─────────────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │    ZooKeeper    │
                        │   (配置分发)     │
                        └────────┬────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
     ┌─────────────────┐┌─────────────────┐┌─────────────────┐
     │  YAF 容器 1     ││  YAF 容器 2     ││  YAF 容器 N     │
     │  ┌───────────┐  ││  ┌───────────┐  ││  ┌───────────┐  │
     │  │Config Agent│ ││  │Config Agent│ ││  │Config Agent│ │
     │  └─────┬─────┘  ││  └─────┬─────┘  ││  └─────┬─────┘  │
     │        │        ││        │        ││        │        │
     │        ▼        ││        ▼        ││        ▼        │
     │  ┌───────────┐  ││  ┌───────────┐  ││  ┌───────────┐  │
     │  │    YAF    │  ││  │    YAF    │  ││  │    YAF    │  │
     │  └───────────┘  ││  └───────────┘  ││  └───────────┘  │
     └─────────────────┘└─────────────────┘└─────────────────┘
```

## 项目结构

```
yf-web/
├── backend/                 # Go 后端服务
│   ├── cmd/server/         # 主程序入口
│   ├── internal/
│   │   ├── api/            # HTTP API 处理器
│   │   ├── db/             # 数据库操作
│   │   ├── models/         # 数据模型
│   │   ├── validator/      # 配置验证
│   │   └── zk/             # ZooKeeper 客户端
│   ├── config.yaml         # 配置文件
│   └── go.mod
├── config-agent/            # 容器内配置代理
│   ├── cmd/agent/          # 主程序入口
│   ├── internal/
│   │   ├── config/         # 配置模型
│   │   ├── supervisor/     # Supervisor 控制
│   │   ├── template/       # 配置模板渲染
│   │   └── watcher/        # ZK 监听器
│   └── go.mod
├── frontend/                # Vue 前端
│   ├── src/
│   │   ├── api/            # API 调用
│   │   ├── components/     # 组件
│   │   ├── router/         # 路由
│   │   ├── stores/         # Pinia 状态
│   │   ├── styles/         # 样式
│   │   └── views/          # 页面
│   └── package.json
├── docker/                  # Docker 配置
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   ├── Dockerfile.agent
│   ├── nginx.conf
│   └── supervisor/
├── docker-compose.yml       # 生产环境部署
└── docker-compose.dev.yml   # 开发环境依赖
```

## 快速开始

### 1. 启动开发环境依赖

```bash
# 启动 PostgreSQL 和 ZooKeeper
docker-compose -f docker-compose.dev.yml up -d
```

### 2. 启动后端服务

```bash
cd backend

# 下载依赖
go mod download

# 运行服务
go run ./cmd/server
```

后端服务默认运行在 `http://localhost:8080`

### 3. 启动前端开发服务器

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

前端开发服务器运行在 `http://localhost:3000`

## 生产环境部署

### 使用 Docker Compose

```bash
# 构建并启动所有服务
docker-compose up -d --build

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

服务访问：
- 前端 UI: http://localhost:3000
- 后端 API: http://localhost:8080
- ZooKeeper: localhost:2181
- PostgreSQL: localhost:5432

## 配置说明

### 后端配置 (backend/config.yaml)

```yaml
server:
  port: 8080
  mode: debug  # debug / release

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: yaf_config
  sslmode: disable

zookeeper:
  servers: localhost:2181
```

也可以通过环境变量配置（格式：`大写_下划线`，如 `DATABASE_HOST`）

### Config Agent 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `ZK_SERVERS` | ZooKeeper 服务器地址 | `localhost:2181` |
| `YAF_CLUSTER` | 集群名称 | `default` |
| `YAF_NODE_ID` | 节点 ID | `node-1` |
| `YAF_CONFIG_PATH` | 配置文件路径 | `/etc/yaf/yaf.init` |
| `YAF_INTERFACE` | 网卡名称 | `eth0` |
| `SM_LISTEN_PORT` | super_mediator 监听端口 | `18000` |
| `FTP_HOST` | FTP 服务器地址 | - |
| `FTP_PORT` | FTP 端口 | `21` |
| `FTP_USER` | FTP 用户名 | - |
| `FTP_PASS` | FTP 密码 | - |
| `FTP_DIR` | FTP 目录 | `/` |

## API 接口

### 全局配置

- `GET /api/v1/config/global` - 获取全局配置
- `POST /api/v1/config/global` - 保存全局配置
- `GET /api/v1/config/global/history` - 获取全局配置历史

### 集群配置

- `GET /api/v1/clusters` - 列出所有集群
- `GET /api/v1/config/cluster/:cluster` - 获取集群配置
- `POST /api/v1/config/cluster/:cluster` - 保存集群配置
- `GET /api/v1/config/cluster/:cluster/history` - 获取集群配置历史

### 节点配置

- `GET /api/v1/clusters/:cluster/nodes` - 列出集群下的节点
- `GET /api/v1/config/cluster/:cluster/node/:node` - 获取节点配置
- `POST /api/v1/config/cluster/:cluster/node/:node` - 保存节点配置
- `GET /api/v1/config/cluster/:cluster/node/:node/history` - 获取节点配置历史

### 其他

- `GET /api/v1/fields` - 获取支持的输出字段列表
- `GET /api/v1/config/default` - 获取默认配置
- `POST /api/v1/config/rollback` - 回滚配置

## ZooKeeper 节点设计

```
/yaf-config/
├── global/
│   └── config              # 全局配置 JSON
└── cluster/
    ├── {cluster-name}/
    │   ├── config          # 集群配置 JSON
    │   └── nodes/
    │       └── {node-id}/
    │           └── config  # 节点配置 JSON
    └── ...
```

## 配置合并策略

配置按以下顺序合并，后者覆盖前者：

1. **默认配置** → 2. **全局配置** → 3. **集群配置** → 4. **节点配置**

每个级别的配置可以只包含需要覆盖的字段。

## 配置模型

```json
{
  "capture": {
    "time_window_ms": 60000,
    "enable_applabel": true,
    "enable_dpi": false,
    "max_payload": 1024
  },
  "filter": {
    "ip_whitelist": ["10.0.0.0/8"],
    "ip_blacklist": ["192.168.1.0/24"],
    "src_ports": [80, 443],
    "dst_ports": [80, 443],
    "bpf_filter": "ip and not port 22"
  },
  "output": {
    "fields": [
      "flowStartMilliseconds",
      "flowEndMilliseconds",
      "sourceIPv4Address",
      "destinationIPv4Address",
      "sourceTransportPort",
      "destinationTransportPort",
      "protocolIdentifier",
      "silkAppLabel"
    ]
  }
}
```

## 在 YAF 容器中集成 Config Agent

1. 将编译好的 `yaf-config-agent` 复制到容器中
2. 更新 supervisor 配置添加 config-agent 程序
3. 设置必要的环境变量

示例 supervisor 配置：

```ini
[program:config-agent]
command=/usr/local/bin/yaf-config-agent
autorestart=true
redirect_stderr=true
stdout_logfile=/var/log/supervisor/config-agent.log
environment=ZK_SERVERS="zk1:2181,zk2:2181",YAF_CLUSTER="production",YAF_NODE_ID="%(ENV_HOSTNAME)s"
```

## 许可证

MIT License

