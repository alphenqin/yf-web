package zk

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
	"go.uber.org/zap"
)

const (
	// 配置路径前缀
	ConfigBasePath = "/xnta/yaf-config"
	GlobalPath     = "/xnta/yaf-config/global/config"
	ClusterPath    = "/xnta/yaf-config/cluster"
)

// Client ZooKeeper 客户端封装
type Client struct {
	conn    *zk.Conn
	servers []string
	logger  *zap.Logger
	mu      sync.RWMutex
}

// Reconnect 重新连接到新的 ZooKeeper 服务器
func (c *Client) Reconnect(servers []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 关闭旧连接
	if c.conn != nil {
		c.conn.Close()
	}

	// 建立新连接
	conn, eventCh, err := zk.Connect(servers, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to zookeeper: %w", err)
	}

	c.conn = conn
	c.servers = servers

	// 监听连接事件
	go c.watchConnection(eventCh)

	c.logger.Info("zookeeper reconnected", zap.Strings("servers", servers))
	return nil
}

// NewClient 创建 ZK 客户端
func NewClient(servers []string, logger *zap.Logger) (*Client, error) {
	conn, eventCh, err := zk.Connect(servers, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to zookeeper: %w", err)
	}

	client := &Client{
		conn:    conn,
		servers: servers,
		logger:  logger,
	}

	// 监听连接事件
	go client.watchConnection(eventCh)

	return client, nil
}

// watchConnection 监听连接状态
func (c *Client) watchConnection(eventCh <-chan zk.Event) {
	for event := range eventCh {
		c.logger.Info("zk connection event",
			zap.String("type", event.Type.String()),
			zap.String("state", event.State.String()),
		)
	}
}

// Close 关闭连接
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
	}
}

// EnsurePath 确保路径存在
func (c *Client) EnsurePath(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	parts := strings.Split(path, "/")
	currentPath := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		currentPath += "/" + part
		exists, _, err := c.conn.Exists(currentPath)
		if err != nil {
			return fmt.Errorf("failed to check path %s: %w", currentPath, err)
		}
		if !exists {
			_, err = c.conn.Create(currentPath, []byte{}, 0, zk.WorldACL(zk.PermAll))
			if err != nil && err != zk.ErrNodeExists {
				return fmt.Errorf("failed to create path %s: %w", currentPath, err)
			}
		}
	}
	return nil
}

// SetConfig 设置配置
func (c *Client) SetConfig(path string, data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 确保路径存在
	parentPath := path[:strings.LastIndex(path, "/")]
	if err := c.EnsurePath(parentPath); err != nil {
		return err
	}

	exists, stat, err := c.conn.Exists(path)
	if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}

	if exists {
		_, err = c.conn.Set(path, jsonData, stat.Version)
	} else {
		_, err = c.conn.Create(path, jsonData, 0, zk.WorldACL(zk.PermAll))
	}

	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	c.logger.Info("config updated in zookeeper",
		zap.String("path", path),
		zap.Int("size", len(jsonData)),
	)
	return nil
}

// GetConfig 获取配置
func (c *Client) GetConfig(path string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, _, err := c.conn.Get(path)
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return data, nil
}

// DeleteConfig 删除配置
func (c *Client) DeleteConfig(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	exists, stat, err := c.conn.Exists(path)
	if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}
	if !exists {
		return nil
	}

	return c.conn.Delete(path, stat.Version)
}

// GetGlobalConfigPath 获取全局配置路径
func GetGlobalConfigPath() string {
	return GlobalPath
}

// GetClusterConfigPath 获取集群配置路径
func GetClusterConfigPath(clusterName string) string {
	return fmt.Sprintf("%s/%s/config", ClusterPath, clusterName)
}

// GetNodeConfigPath 获取节点配置路径
func GetNodeConfigPath(clusterName, nodeID string) string {
	return fmt.Sprintf("%s/%s/nodes/%s/config", ClusterPath, clusterName, nodeID)
}

// ListClusters 列出所有集群
func (c *Client) ListClusters() ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	children, _, err := c.conn.Children(ClusterPath)
	if err != nil {
		if err == zk.ErrNoNode {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	return children, nil
}

// ListNodes 列出集群下所有节点
func (c *Client) ListNodes(clusterName string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodesPath := fmt.Sprintf("%s/%s/nodes", ClusterPath, clusterName)
	children, _, err := c.conn.Children(nodesPath)
	if err != nil {
		if err == zk.ErrNoNode {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	return children, nil
}

// IsConnected 检查 ZK 是否连接
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return false
	}
	state := c.conn.State()
	return state == zk.StateConnected || state == zk.StateHasSession
}

// GetState 获取连接状态字符串
func (c *Client) GetState() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return "disconnected"
	}
	return c.conn.State().String()
}

// GetServers 获取服务器列表
func (c *Client) GetServers() []string {
	return c.servers
}

