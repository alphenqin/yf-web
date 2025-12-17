package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/yf-web/config-agent/internal/config"
	"github.com/yf-web/config-agent/internal/supervisor"
	"github.com/yf-web/config-agent/internal/template"
	"github.com/yf-web/config-agent/internal/watcher"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// 初始化日志
	logger := initLogger()
	defer logger.Sync()

	logger.Info("yaf-config-agent starting...")

	// 读取环境变量
	zkServers := getEnv("ZK_SERVERS", "localhost:2181")
	cluster := getEnv("YAF_CLUSTER", "default")
	nodeID := getEnv("YAF_NODE_ID", "node-1")
	configPath := getEnv("YAF_CONFIG_PATH", "/etc/yaf/yaf.init")
	interface_ := getEnv("YAF_INTERFACE", "eth0")
	ipfixPort := getEnv("SM_LISTEN_PORT", "18000")

	logger.Info("configuration",
		zap.String("zk_servers", zkServers),
		zap.String("cluster", cluster),
		zap.String("node_id", nodeID),
		zap.String("config_path", configPath),
		zap.String("interface", interface_),
	)

	// 创建 supervisor 控制器
	superCtrl := supervisor.NewController(logger)

	// 检查 supervisord 是否运行
	if !superCtrl.CheckSupervisor() {
		logger.Warn("supervisord not detected, restart commands will fail")
	}

	// 创建模板生成器
	generator, err := template.NewGenerator(
		configPath, interface_, ipfixPort,
		cluster, nodeID,
		logger,
	)
	if err != nil {
		logger.Fatal("failed to create template generator", zap.Error(err))
	}

	// 配置变更回调
	onConfigChange := func(cfg *config.YafConfig) error {
		logger.Info("applying new configuration",
			zap.String("interface", cfg.Capture.Interface),
			zap.Bool("applabel", cfg.Capture.EnableAppLabel),
			zap.Int("output_fields", len(cfg.Output.Fields)),
		)

		// 生成配置文件
		if err := generator.Generate(cfg); err != nil {
			logger.Error("failed to generate config file", zap.Error(err))
			return err
		}

		// 重启 YAF 和 Pipeline（pipeline 包含 super_mediator + processor）
		if err := superCtrl.RestartAll(); err != nil {
			logger.Error("failed to restart processes", zap.Error(err))
			// 不返回错误，配置已写入
		}

		return nil
	}

	// 创建配置监听器
	servers := strings.Split(zkServers, ",")
	configWatcher, err := watcher.NewConfigWatcher(servers, cluster, nodeID, logger, onConfigChange)
	if err != nil {
		logger.Fatal("failed to create config watcher", zap.Error(err))
	}

	// 启动监听
	if err := configWatcher.Start(); err != nil {
		logger.Fatal("failed to start config watcher", zap.Error(err))
	}

	logger.Info("config-agent started, watching for configuration changes...")

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down config-agent...")
	configWatcher.Stop()
}

func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.OutputPaths = []string{"stdout"}
	logger, _ := config.Build()
	return logger
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
