# yaf-processor

yaf-processor 是一个 YAF（Yet Another Flowmeter）数据处理器，用于处理 YAF 生成的流量数据。

## 项目简介

yaf-processor 通过 Docker 容器化部署，实现了从网络流量采集到数据处理的完整流程：

1. **YAF**：从网络接口抓取流量，生成 IPFIX 流记录
2. **super_mediator**：将 IPFIX 转换为 CSV 文本格式
3. **processor**：接收 CSV 数据，滚动写入本地 gzip 压缩文件，并支持状态上报

### processor 组件

processor 是项目的核心组件，使用 Go 语言开发，负责：
- 从标准输入（管道）读取 CSV 数据
- 滚动写入本地 gzip 压缩文件（按时间和大小策略）
- 定期上报运行状态信息
- 预留输出接口，方便未来对接 Kafka、MQ 等消息队列
- 所有配置通过 `yaf.init` 配置文件统一管理

## 功能特性

### 整体方案
- **一体化部署**：YAF + super_mediator + processor 集成在一个 Docker 镜像中
- **开箱即用**：容器启动后自动运行完整的流量采集和处理流程
- **配置统一**：所有组件配置集中在 `yaf.init` 文件中

### processor 组件
- **滚动文件写入**：按时间间隔和文件大小自动滚动生成新的压缩文件
- **状态上报**：定期上报运行状态信息到配置的 URL
- **配置驱动**：所有参数通过 `yaf.init` 配置文件中的 `processor` 配置块管理
- **优雅关闭**：支持信号处理，确保数据不丢失
- **可扩展输出**：预留 Sink 接口，方便未来对接 Kafka、MQ 等

## 项目结构

```
yaf-processor/
├── cmd/
│   └── processor/          # 主程序入口
│       └── main.go
├── internal/
│   ├── config/            # 配置解析模块
│   │   └── config.go
│   ├── converter/         # 时间转换模块
│   │   └── time_converter.go
│   ├── reporter/         # 状态上报模块
│   │   └── reporter.go
│   ├── sink/             # 输出接口（预留）
│   │   └── sink.go
│   └── writer/           # 文件滚动写入模块
│       └── writer.go
├── go.mod
├── go.sum
└── README.md
```

## 构建

需要 Go 1.21+

```bash
# 拉取依赖
go mod tidy

# 编译（当前平台）
go build -o processor ./cmd/processor

# 编译 Linux amd64（用于 Docker）
GOOS=linux GOARCH=amd64 go build -o processor ./cmd/processor
```

## 使用方法

### 命令行参数

- `-config string`：**必填**。YAF 配置文件路径（`yaf.init`），例如：`/etc/yaf/yaf.init`
- `-data-dir string`：**必填**。本地缓存目录，用来存放滚动生成的压缩文件
- `-log-level string`：可选。日志级别：`debug|info|warn|error`，默认 `info`

### 配置文件格式

所有配置都配置在 `yaf.init` 文件的 `processor` 配置块中：

```lua
-- YAF 的原始配置
input = {
    type = "pcap",
    inf = "any"
}

output = {
    host = "127.0.0.1",
    port = "18000",
    protocol = "tcp"
}

-- ... 其他 YAF 配置 ...

--------------------------------------------------------
-- processor 配置块
--------------------------------------------------------
processor = {
    -- 本地滚动文件策略
    rotate_interval_sec = 60,  -- 每 60s 强制切新文件
    rotate_size_mb      = 100,  -- 单文件最大 100 MiB
    file_prefix         = "flows_",  -- flows_YYYYMMDD_HHMMSS_xxx.csv.gz
    timezone            = "Asia/Shanghai",  -- 时区

    -- 输出类型（file/kafka/mq等，未来扩展）
    output_type = "file",

    -- 状态上报配置
    status_report_url = "http://example.com/api/uploadStatus",
    status_report_interval_sec = 60,  -- 每 60s 上报一次状态
    uuid = "container-hostname",  -- 容器主机名（可选）
}
```

### 运行方式

程序从**标准输入（stdin）**读取 CSV 数据，支持以下使用场景：

1. **管道输入**（推荐）：与其他工具通过管道连接
2. **重定向输入**：从文件重定向到标准输入
3. **Docker 容器内**：作为 super_mediator 的下游处理程序

## 状态上报

processor 会定期上报运行状态信息，包括：

- 当前收到的包数/字节数
- 当前处理的包数/字节数
- 当前平均值（每秒）
- 总计收到的包数/字节数
- 总计处理的包数/字节数
- 总计平均值（每秒）
- 运行时间
- 容器主机名

上报数据以 JSON 格式 POST 到配置的 URL。

## 输出接口（Sink）

processor 预留了 Sink 接口，方便未来对接不同的输出目标：

- **Sink**：基础输出接口
- **StreamSink**：流式输出接口
- **BatchSink**：批量输出接口

未来可以实现：
- Kafka Sink
- MQ Sink
- 其他消息队列 Sink

## 文件命名规则

生成的文件命名格式：`{file_prefix}{YYYYMMDD_HHMMSS}_{index}.csv.gz`

例如：
- `flows_20251127_102030_000.csv.gz`
- `flows_20251127_102030_001.csv.gz`

写入时使用 `.part` 后缀，完成后自动重命名为 `.csv.gz`：
- 写入中：`flows_20251127_102030_000.part`
- 完成后：`flows_20251127_102030_000.csv.gz`

## 滚动策略

文件会在以下任一条件满足时滚动：

1. **时间间隔**：自文件创建起超过 `rotate_interval_sec` 秒
2. **文件大小**：已写入的原始字节数（未压缩）超过 `rotate_size_mb` MB

## 日志输出

所有日志输出到 `stderr`，包含时间戳：

```
2025/11/27 10:00:00 [INFO] 配置加载成功: 滚动间隔=60s, 滚动大小=100MB, 输出类型=file
2025/11/27 10:00:00 [INFO] 数据目录已就绪: /data
2025/11/27 10:00:00 [INFO] 状态上报器已启动，上报间隔: 60s
2025/11/27 10:10:00 [INFO] 状态上报成功: 运行时间=600s, 总接收=10000包/1024000字节, 总处理=10000包/1024000字节
```
