package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/yf-web/backend/internal/api"
	"github.com/yf-web/backend/internal/db"
	"github.com/yf-web/backend/internal/zk"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// 初始化日志
	logger := initLogger()
	defer logger.Sync()

	// 加载配置
	loadConfig()

	// 连接数据库
	dbConfig := db.Config{
		Host:     viper.GetString("database.host"),
		Port:     viper.GetInt("database.port"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
		DBName:   viper.GetString("database.dbname"),
		SSLMode:  viper.GetString("database.sslmode"),
	}
	database, err := db.NewPostgresDB(dbConfig, logger)
	if err != nil {
		logger.Fatal("failed to connect database", zap.Error(err))
	}
	defer database.Close()
	logger.Info("database connected")

	// 连接 ZooKeeper
	zkServers := strings.Split(viper.GetString("zookeeper.servers"), ",")
	zkClient, err := zk.NewClient(zkServers, logger)
	if err != nil {
		logger.Fatal("failed to connect zookeeper", zap.Error(err))
	}
	defer zkClient.Close()
	logger.Info("zookeeper connected", zap.Strings("servers", zkServers))

	// 确保 ZK 基础路径存在
	if err := zkClient.EnsurePath(zk.ConfigBasePath + "/global"); err != nil {
		logger.Warn("failed to ensure zk path", zap.Error(err))
	}
	if err := zkClient.EnsurePath(zk.ClusterPath); err != nil {
		logger.Warn("failed to ensure zk cluster path", zap.Error(err))
	}

	// 创建 API 处理器
	handler := api.NewHandler(database, zkClient, logger)

	// 设置 Gin
	if viper.GetString("server.mode") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(ginLogger(logger))

	// 注册路由
	handler.RegisterRoutes(r)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 启动服务器
	addr := fmt.Sprintf(":%d", viper.GetInt("server.port"))
	logger.Info("starting server", zap.String("addr", addr))

	go func() {
		if err := r.Run(addr); err != nil {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")
}

func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := config.Build()
	return logger
}

func loadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".") // 当前目录
	viper.AddConfigPath("/etc/yaf-config-service/")

	// 默认配置
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "yaf_config")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("zookeeper.servers", "localhost:2181")

	// 支持环境变量
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Error reading config file: %v\n", err)
		}
	}
}

func ginLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		logger.Info("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("ip", c.ClientIP()),
		)
	}
}
