package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	logger.Info("configuration",
		zap.String("zk_servers", zkServers),
		zap.String("cluster", cluster),
		zap.String("node_id", nodeID),
		zap.String("config_path", configPath),
	)

	// 创建 supervisor 控制器
	superCtrl := supervisor.NewController(logger)

	// 检查 supervisord 是否运行
	if !superCtrl.CheckSupervisor() {
		logger.Warn("supervisord not detected, restart commands will fail")
	}

	// 创建模板生成器（interface 和 ipfixPort 使用硬编码默认值）
	generator, err := template.NewGenerator(
		configPath,
		cluster, nodeID,
		logger,
	)
	if err != nil {
		logger.Fatal("failed to create template generator", zap.Error(err))
	}

	// 配置变更回调
	onConfigChange := func(cfg *config.YafConfig) error {
		applyStartTime := time.Now()
		logger.Info("[CONFIG_APPLY] 收到配置变更，开始应用新配置",
			zap.Time("apply_time", applyStartTime),
			zap.String("interface", cfg.Capture.Interface),
			zap.Int("ipfix_port", cfg.Capture.IPFIXPort),
			zap.Bool("enable_applabel", cfg.Capture.EnableAppLabel),
			zap.Bool("enable_dpi", cfg.Capture.EnableDPI),
			zap.Int("max_payload", cfg.Capture.MaxPayload),
			zap.Int("output_fields_count", len(cfg.Output.Fields)),
			zap.Strings("output_fields", cfg.Output.Fields),
		)

		// 生成配置文件
		generateStartTime := time.Now()
		if err := generator.Generate(cfg); err != nil {
			logger.Error("[CONFIG_APPLY] 配置文件生成失败",
				zap.Error(err),
				zap.Duration("generate_duration", time.Since(generateStartTime)),
			)
			return err
		}
		logger.Info("[CONFIG_APPLY] 配置文件生成成功",
			zap.String("config_path", configPath),
			zap.Duration("generate_duration", time.Since(generateStartTime)),
		)

		// 重启 YAF 和 Pipeline（pipeline 包含 super_mediator + processor）
		restartStartTime := time.Now()
		logger.Info("[CONFIG_APPLY] 开始重启进程以应用新配置",
			zap.Time("restart_time", restartStartTime),
		)
		if err := superCtrl.RestartAll(); err != nil {
			logger.Error("[CONFIG_APPLY] 进程重启失败",
				zap.Error(err),
				zap.Duration("restart_duration", time.Since(restartStartTime)),
				zap.Duration("total_duration", time.Since(applyStartTime)),
			)
			// 不返回错误，配置已写入
		} else {
			logger.Info("[CONFIG_APPLY] 配置应用完成",
				zap.Duration("restart_duration", time.Since(restartStartTime)),
				zap.Duration("total_duration", time.Since(applyStartTime)),
			)
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
