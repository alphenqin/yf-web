package watcher

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/yf-web/config-agent/internal/config"
	"go.uber.org/zap"
)

const (
	// 配置路径前缀（需与后端保持一致）
	ConfigBasePath = "/xnta/yaf-config"
	GlobalPath     = "/xnta/yaf-config/global/config"
	ClusterPath    = "/xnta/yaf-config/cluster"
)

// ConfigWatcher ZK 配置监听器
type ConfigWatcher struct {
	conn        *zk.Conn
	cluster     string
	nodeID      string
	logger      *zap.Logger
	onChange    func(*config.YafConfig) error
	stopCh      chan struct{}
	mu          sync.RWMutex
	lastConfig  *config.YafConfig
}

// NewConfigWatcher 创建配置监听器
func NewConfigWatcher(servers []string, cluster, nodeID string, logger *zap.Logger, onChange func(*config.YafConfig) error) (*ConfigWatcher, error) {
	conn, eventCh, err := zk.Connect(servers, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to zookeeper: %w", err)
	}

	watcher := &ConfigWatcher{
		conn:     conn,
		cluster:  cluster,
		nodeID:   nodeID,
		logger:   logger,
		onChange: onChange,
		stopCh:   make(chan struct{}),
	}

	// 监听连接事件
	go watcher.handleConnectionEvents(eventCh)

	return watcher, nil
}

// handleConnectionEvents 处理连接事件
func (w *ConfigWatcher) handleConnectionEvents(eventCh <-chan zk.Event) {
	for {
		select {
		case event := <-eventCh:
			w.logger.Info("zk connection event",
				zap.String("type", event.Type.String()),
				zap.String("state", event.State.String()),
			)
			if event.State == zk.StateHasSession {
				// 重新连接后重新加载配置
				go w.loadAndApplyConfig()
			}
		case <-w.stopCh:
			return
		}
	}
}

// Start 启动监听
func (w *ConfigWatcher) Start() error {
	// 首次加载配置
	if err := w.loadAndApplyConfig(); err != nil {
		w.logger.Error("initial config load failed", zap.Error(err))
		// 不返回错误，继续监听
	}

	// 启动监听循环
	go w.watchLoop()

	return nil
}

// Stop 停止监听
func (w *ConfigWatcher) Stop() {
	close(w.stopCh)
	if w.conn != nil {
		w.conn.Close()
	}
}

// watchLoop 监听循环
func (w *ConfigWatcher) watchLoop() {
	for {
		select {
		case <-w.stopCh:
			return
		default:
		}

		// 获取所有配置路径的 watch
		globalPath := GlobalPath
		clusterPath := fmt.Sprintf("%s/%s/config", ClusterPath, w.cluster)
		nodePath := fmt.Sprintf("%s/%s/nodes/%s/config", ClusterPath, w.cluster, w.nodeID)

		// 创建 watch channels
		var globalCh, clusterCh, nodeCh <-chan zk.Event

		// Global config watch
		_, _, globalCh, err := w.conn.GetW(globalPath)
		if err != nil && err != zk.ErrNoNode {
			w.logger.Warn("failed to watch global config", zap.Error(err))
		}

		// Cluster config watch
		_, _, clusterCh, err = w.conn.GetW(clusterPath)
		if err != nil && err != zk.ErrNoNode {
			w.logger.Warn("failed to watch cluster config", zap.Error(err))
		}

		// Node config watch
		_, _, nodeCh, err = w.conn.GetW(nodePath)
		if err != nil && err != zk.ErrNoNode {
			w.logger.Warn("failed to watch node config", zap.Error(err))
		}

		// 等待任意一个配置变化
		select {
		case <-w.stopCh:
			return
		case event := <-globalCh:
			w.logger.Info("[CONFIG_CHANGE] 检测到全局配置变更",
				zap.String("event_type", event.Type.String()),
				zap.String("event_state", event.State.String()),
				zap.String("source", "global"),
				zap.String("path", globalPath),
			)
		case event := <-clusterCh:
			w.logger.Info("[CONFIG_CHANGE] 检测到集群配置变更",
				zap.String("event_type", event.Type.String()),
				zap.String("event_state", event.State.String()),
				zap.String("source", "cluster"),
				zap.String("cluster", w.cluster),
				zap.String("path", clusterPath),
			)
		case event := <-nodeCh:
			w.logger.Info("[CONFIG_CHANGE] 检测到节点配置变更",
				zap.String("event_type", event.Type.String()),
				zap.String("event_state", event.State.String()),
				zap.String("source", "node"),
				zap.String("cluster", w.cluster),
				zap.String("node_id", w.nodeID),
				zap.String("path", nodePath),
			)
		case <-time.After(30 * time.Second):
			// 定期刷新 watch（防止 session 过期）
			continue
		}

		// 重新加载配置
		if err := w.loadAndApplyConfig(); err != nil {
			w.logger.Error("failed to reload config", zap.Error(err))
		}
	}
}

// loadAndApplyConfig 加载并应用配置
func (w *ConfigWatcher) loadAndApplyConfig() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	startTime := time.Now()
	w.logger.Info("[CONFIG_LOAD] 开始加载配置",
		zap.String("cluster", w.cluster),
		zap.String("node_id", w.nodeID),
	)

	// 加载各级配置
	globalCfg := w.loadConfig(GlobalPath)
	clusterPath := fmt.Sprintf("%s/%s/config", ClusterPath, w.cluster)
	clusterCfg := w.loadConfig(clusterPath)
	nodePath := fmt.Sprintf("%s/%s/nodes/%s/config", ClusterPath, w.cluster, w.nodeID)
	nodeCfg := w.loadConfig(nodePath)

	// 合并配置：global → cluster → node
	merged := config.DefaultConfig()
	merged = config.MergeConfig(merged, globalCfg)
	merged = config.MergeConfig(merged, clusterCfg)
	merged = config.MergeConfig(merged, nodeCfg)

	w.logger.Info("[CONFIG_LOAD] 配置加载完成",
		zap.Bool("has_global", globalCfg != nil),
		zap.Bool("has_cluster", clusterCfg != nil),
		zap.Bool("has_node", nodeCfg != nil),
		zap.String("interface", merged.Capture.Interface),
		zap.Int("ipfix_port", merged.Capture.IPFIXPort),
		zap.Bool("enable_applabel", merged.Capture.EnableAppLabel),
		zap.Int("output_fields_count", len(merged.Output.Fields)),
		zap.Duration("load_duration", time.Since(startTime)),
	)

	// 检查配置是否变化
	if w.configEqual(w.lastConfig, merged) {
		w.logger.Info("[CONFIG_LOAD] 配置未变化，跳过应用",
			zap.Duration("check_duration", time.Since(startTime)),
		)
		return nil
	}

	w.logger.Info("[CONFIG_LOAD] 检测到配置变化，准备应用新配置",
		zap.Duration("check_duration", time.Since(startTime)),
	)

	w.lastConfig = merged

	// 调用回调应用配置
	applyStartTime := time.Now()
	if w.onChange != nil {
		if err := w.onChange(merged); err != nil {
			w.logger.Error("[CONFIG_APPLY] 配置应用失败",
				zap.Error(err),
				zap.Duration("apply_duration", time.Since(applyStartTime)),
			)
			return fmt.Errorf("failed to apply config: %w", err)
		}
		w.logger.Info("[CONFIG_APPLY] 配置应用成功",
			zap.Duration("apply_duration", time.Since(applyStartTime)),
			zap.Duration("total_duration", time.Since(startTime)),
		)
	}

	return nil
}

// loadConfig 从 ZK 加载配置
func (w *ConfigWatcher) loadConfig(path string) *config.YafConfig {
	data, _, err := w.conn.Get(path)
	if err != nil {
		if err != zk.ErrNoNode {
			w.logger.Warn("failed to get config", zap.String("path", path), zap.Error(err))
		}
		return nil
	}

	var cfg config.YafConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		w.logger.Warn("failed to parse config", zap.String("path", path), zap.Error(err))
		return nil
	}

	return &cfg
}

// configEqual 比较两个配置是否相等
func (w *ConfigWatcher) configEqual(a, b *config.YafConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// 简单比较 JSON
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

