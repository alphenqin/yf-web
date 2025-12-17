package sink

import (
	"context"
	"io"
)

// Sink 数据输出接口，用于对接不同的输出目标（Kafka、MQ、FTP等）
type Sink interface {
	// Send 发送数据到输出目标
	// ctx: 上下文，用于取消操作
	// data: 要发送的数据
	// 返回错误如果发送失败
	Send(ctx context.Context, data []byte) error

	// SendFile 发送文件到输出目标
	// ctx: 上下文，用于取消操作
	// filePath: 文件路径
	// 返回错误如果发送失败
	SendFile(ctx context.Context, filePath string) error

	// Start 启动 Sink（如果需要后台运行）
	Start() error

	// Stop 停止 Sink
	Stop() error

	// Close 关闭 Sink 并释放资源
	Close() error
}

// StreamSink 流式输出接口，用于需要流式传输的场景
type StreamSink interface {
	Sink

	// OpenStream 打开一个流，返回 Writer
	// ctx: 上下文
	// streamID: 流标识符（可选，用于区分不同的流）
	// 返回 Writer 和错误
	OpenStream(ctx context.Context, streamID string) (io.WriteCloser, error)
}

// BatchSink 批量输出接口，用于批量发送的场景
type BatchSink interface {
	Sink

	// SendBatch 批量发送数据
	// ctx: 上下文
	// batch: 批量数据
	// 返回错误如果发送失败
	SendBatch(ctx context.Context, batch [][]byte) error
}

