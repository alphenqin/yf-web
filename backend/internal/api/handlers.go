package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yf-web/backend/internal/db"
	"github.com/yf-web/backend/internal/models"
	"github.com/yf-web/backend/internal/validator"
	"github.com/yf-web/backend/internal/zk"
	"go.uber.org/zap"
)

// Handler API 处理器
type Handler struct {
	db        *db.PostgresDB
	zkClient  *zk.Client
	validator *validator.ConfigValidator
	logger    *zap.Logger
}

// NewHandler 创建处理器
func NewHandler(db *db.PostgresDB, zkClient *zk.Client, logger *zap.Logger) *Handler {
	return &Handler{
		db:        db,
		zkClient:  zkClient,
		validator: validator.NewConfigValidator(),
		logger:    logger,
	}
}

// Response 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ConfigRequest 配置请求
type ConfigRequest struct {
	Config    models.YafConfig `json:"config"`
	CreatedBy string           `json:"created_by"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SettingsRequest 系统设置请求
type SettingsRequest struct {
	ZookeeperServers string `json:"zookeeper_servers"`
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// CORS 中间件
	r.Use(corsMiddleware())

	api := r.Group("/api/v1")
	{
		// 登录接口
		api.POST("/auth/login", h.Login)

		// 系统设置
		api.GET("/settings", h.GetSettings)
		api.POST("/settings", h.SaveSettings)

		// 系统状态
		api.GET("/status", h.GetSystemStatus)

		// 获取支持的字段列表
		api.GET("/fields", h.GetSupportedFields)

		// 获取默认配置
		api.GET("/config/default", h.GetDefaultConfig)

		// 全局配置
		api.GET("/config/global", h.GetGlobalConfig)
		api.POST("/config/global", h.SaveGlobalConfig)
		api.GET("/config/global/history", h.GetGlobalConfigHistory)

		// 集群配置
		api.GET("/clusters", h.ListClusters)
		api.GET("/config/cluster/:cluster", h.GetClusterConfig)
		api.POST("/config/cluster/:cluster", h.SaveClusterConfig)
		api.GET("/config/cluster/:cluster/history", h.GetClusterConfigHistory)

		// 节点配置
		api.GET("/clusters/:cluster/nodes", h.ListNodes)
		api.GET("/config/cluster/:cluster/node/:node", h.GetNodeConfig)
		api.POST("/config/cluster/:cluster/node/:node", h.SaveNodeConfig)
		api.GET("/config/cluster/:cluster/node/:node/history", h.GetNodeConfigHistory)

		// 配置回滚
		api.POST("/config/rollback", h.RollbackConfig)
	}
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "用户名和密码不能为空"})
		return
	}

	valid, err := h.db.ValidateUser(req.Username, req.Password)
	if err != nil {
		h.logger.Error("failed to validate user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: "服务器错误"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, Response{Code: 401, Message: "用户名或密码错误"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "登录成功",
		Data: map[string]string{
			"username": req.Username,
			"token":    "logged_in",
		},
	})
}

// GetSettings 获取系统设置
func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.db.GetAllSettings()
	if err != nil {
		h.logger.Error("failed to get settings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}

	// 如果数据库中没有设置，返回当前运行时的值
	if settings == nil {
		settings = make(map[string]string)
	}
	if _, ok := settings["zookeeper_servers"]; !ok {
		settings["zookeeper_servers"] = strings.Join(h.zkClient.GetServers(), ",")
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    settings,
	})
}

// SaveSettings 保存系统设置
func (h *Handler) SaveSettings(c *gin.Context) {
	var req SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	// 验证 ZooKeeper 地址格式
	if req.ZookeeperServers == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "ZooKeeper 地址不能为空"})
		return
	}

	// 保存到数据库
	if err := h.db.SetSetting("zookeeper_servers", req.ZookeeperServers); err != nil {
		h.logger.Error("failed to save settings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}

	// 尝试重新连接 ZooKeeper
	servers := strings.Split(req.ZookeeperServers, ",")
	for i := range servers {
		servers[i] = strings.TrimSpace(servers[i])
	}

	if err := h.zkClient.Reconnect(servers); err != nil {
		h.logger.Error("failed to reconnect zookeeper", zap.Error(err))
		c.JSON(http.StatusOK, Response{
			Code:    0,
			Message: "设置已保存，但连接 ZooKeeper 失败: " + err.Error(),
			Data:    map[string]bool{"connected": false},
		})
		return
	}

	// 确保 ZK 基础路径存在
	h.zkClient.EnsurePath(zk.ConfigBasePath + "/global")
	h.zkClient.EnsurePath(zk.ClusterPath)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "设置已保存，ZooKeeper 已重新连接",
		Data:    map[string]bool{"connected": true},
	})
}

// GetSystemStatus 获取系统状态
func (h *Handler) GetSystemStatus(c *gin.Context) {
	zkConnected := h.zkClient.IsConnected()
	zkState := h.zkClient.GetState()
	zkServers := h.zkClient.GetServers()

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"zookeeper": map[string]interface{}{
				"connected": zkConnected,
				"state":     zkState,
				"servers":   zkServers,
			},
			"database": map[string]interface{}{
				"connected": true, // 如果能响应请求，说明数据库正常
			},
		},
	})
}

// GetSupportedFields 获取支持的字段列表
func (h *Handler) GetSupportedFields(c *gin.Context) {
	// 返回字段列表和中文名称
	type FieldInfo struct {
		Name  string `json:"name"`
		Label string `json:"label"`
	}
	var fields []FieldInfo
	for _, name := range models.SupportedFields {
		fields = append(fields, FieldInfo{
			Name:  name,
			Label: models.FieldLabels[name],
		})
	}
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    fields,
	})
}

// GetDefaultConfig 获取默认配置
func (h *Handler) GetDefaultConfig(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    models.DefaultConfig(),
	})
}

// GetGlobalConfig 获取全局配置
func (h *Handler) GetGlobalConfig(c *gin.Context) {
	record, err := h.db.GetLatestConfig(models.ScopeGlobal, "", "")
	if err != nil {
		h.logger.Error("failed to get global config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	if record == nil {
		c.JSON(http.StatusOK, Response{Code: 0, Message: "no config found", Data: nil})
		return
	}

	var cfg models.YafConfig
	if err := json.Unmarshal([]byte(record.ConfigJSON), &cfg); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: "invalid config json"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"config":     cfg,
			"version":    record.Version,
			"created_at": record.CreatedAt,
			"created_by": record.CreatedBy,
		},
	})
}

// SaveGlobalConfig 保存全局配置
func (h *Handler) SaveGlobalConfig(c *gin.Context) {
	var req ConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	// 验证配置
	if err := h.validator.Validate(&req.Config); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	configJSON, _ := json.Marshal(req.Config)

	// 保存到数据库
	record := &models.ConfigRecord{
		Scope:      models.ScopeGlobal,
		ConfigJSON: string(configJSON),
		CreatedBy:  req.CreatedBy,
	}
	if err := h.db.SaveConfig(record); err != nil {
		h.logger.Error("failed to save global config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}

	// 同步到 ZooKeeper
	if err := h.zkClient.SetConfig(zk.GetGlobalConfigPath(), req.Config); err != nil {
		h.logger.Error("failed to sync global config to zk", zap.Error(err))
		// 不返回错误，数据库已保存
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    map[string]int{"version": record.Version},
	})
}

// GetGlobalConfigHistory 获取全局配置历史
func (h *Handler) GetGlobalConfigHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	records, err := h.db.GetConfigHistory(models.ScopeGlobal, "", "", limit)
	if err != nil {
		h.logger.Error("failed to get config history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: records})
}

// ListClusters 列出所有集群
func (h *Handler) ListClusters(c *gin.Context) {
	clusters, err := h.db.ListClusters()
	if err != nil {
		h.logger.Error("failed to list clusters", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: clusters})
}

// GetClusterConfig 获取集群配置
func (h *Handler) GetClusterConfig(c *gin.Context) {
	cluster := c.Param("cluster")
	if err := h.validator.ValidateClusterName(cluster); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	record, err := h.db.GetLatestConfig(models.ScopeCluster, cluster, "")
	if err != nil {
		h.logger.Error("failed to get cluster config", zap.Error(err), zap.String("cluster", cluster))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	if record == nil {
		c.JSON(http.StatusOK, Response{Code: 0, Message: "no config found", Data: nil})
		return
	}

	var cfg models.YafConfig
	if err := json.Unmarshal([]byte(record.ConfigJSON), &cfg); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: "invalid config json"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"cluster":    cluster,
			"config":     cfg,
			"version":    record.Version,
			"created_at": record.CreatedAt,
			"created_by": record.CreatedBy,
		},
	})
}

// SaveClusterConfig 保存集群配置
func (h *Handler) SaveClusterConfig(c *gin.Context) {
	cluster := c.Param("cluster")
	if err := h.validator.ValidateClusterName(cluster); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	var req ConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	if err := h.validator.Validate(&req.Config); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	configJSON, _ := json.Marshal(req.Config)

	record := &models.ConfigRecord{
		Scope:       models.ScopeCluster,
		ClusterName: cluster,
		ConfigJSON:  string(configJSON),
		CreatedBy:   req.CreatedBy,
	}
	if err := h.db.SaveConfig(record); err != nil {
		h.logger.Error("failed to save cluster config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}

	if err := h.zkClient.SetConfig(zk.GetClusterConfigPath(cluster), req.Config); err != nil {
		h.logger.Error("failed to sync cluster config to zk", zap.Error(err))
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    map[string]interface{}{"version": record.Version, "cluster": cluster},
	})
}

// GetClusterConfigHistory 获取集群配置历史
func (h *Handler) GetClusterConfigHistory(c *gin.Context) {
	cluster := c.Param("cluster")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	records, err := h.db.GetConfigHistory(models.ScopeCluster, cluster, "", limit)
	if err != nil {
		h.logger.Error("failed to get config history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: records})
}

// ListNodes 列出集群下的节点
func (h *Handler) ListNodes(c *gin.Context) {
	cluster := c.Param("cluster")
	if err := h.validator.ValidateClusterName(cluster); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	nodes, err := h.db.ListNodes(cluster)
	if err != nil {
		h.logger.Error("failed to list nodes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: nodes})
}

// GetNodeConfig 获取节点配置
func (h *Handler) GetNodeConfig(c *gin.Context) {
	cluster := c.Param("cluster")
	node := c.Param("node")

	if err := h.validator.ValidateClusterName(cluster); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.validator.ValidateNodeID(node); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	record, err := h.db.GetLatestConfig(models.ScopeNode, cluster, node)
	if err != nil {
		h.logger.Error("failed to get node config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	if record == nil {
		c.JSON(http.StatusOK, Response{Code: 0, Message: "no config found", Data: nil})
		return
	}

	var cfg models.YafConfig
	if err := json.Unmarshal([]byte(record.ConfigJSON), &cfg); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: "invalid config json"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"cluster":    cluster,
			"node":       node,
			"config":     cfg,
			"version":    record.Version,
			"created_at": record.CreatedAt,
			"created_by": record.CreatedBy,
		},
	})
}

// SaveNodeConfig 保存节点配置
func (h *Handler) SaveNodeConfig(c *gin.Context) {
	cluster := c.Param("cluster")
	node := c.Param("node")

	if err := h.validator.ValidateClusterName(cluster); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.validator.ValidateNodeID(node); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	var req ConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	if err := h.validator.Validate(&req.Config); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	configJSON, _ := json.Marshal(req.Config)

	record := &models.ConfigRecord{
		Scope:       models.ScopeNode,
		ClusterName: cluster,
		NodeID:      node,
		ConfigJSON:  string(configJSON),
		CreatedBy:   req.CreatedBy,
	}
	if err := h.db.SaveConfig(record); err != nil {
		h.logger.Error("failed to save node config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}

	if err := h.zkClient.SetConfig(zk.GetNodeConfigPath(cluster, node), req.Config); err != nil {
		h.logger.Error("failed to sync node config to zk", zap.Error(err))
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    map[string]interface{}{"version": record.Version, "cluster": cluster, "node": node},
	})
}

// GetNodeConfigHistory 获取节点配置历史
func (h *Handler) GetNodeConfigHistory(c *gin.Context) {
	cluster := c.Param("cluster")
	node := c.Param("node")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	records, err := h.db.GetConfigHistory(models.ScopeNode, cluster, node, limit)
	if err != nil {
		h.logger.Error("failed to get config history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: records})
}

// RollbackRequest 回滚请求
type RollbackRequest struct {
	Scope       string `json:"scope"`
	ClusterName string `json:"cluster_name,omitempty"`
	NodeID      string `json:"node_id,omitempty"`
	Version     int    `json:"version"`
	CreatedBy   string `json:"created_by"`
}

// RollbackConfig 回滚配置
func (h *Handler) RollbackConfig(c *gin.Context) {
	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: err.Error()})
		return
	}

	scope := models.ConfigScope(req.Scope)
	record, err := h.db.GetConfigByVersion(scope, req.ClusterName, req.NodeID, req.Version)
	if err != nil {
		h.logger.Error("failed to get config version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}
	if record == nil {
		c.JSON(http.StatusNotFound, Response{Code: 404, Message: "version not found"})
		return
	}

	// 创建新版本（回滚实际上是创建一个内容相同的新版本）
	newRecord := &models.ConfigRecord{
		Scope:       scope,
		ClusterName: req.ClusterName,
		NodeID:      req.NodeID,
		ConfigJSON:  record.ConfigJSON,
		CreatedBy:   req.CreatedBy,
	}
	if err := h.db.SaveConfig(newRecord); err != nil {
		h.logger.Error("failed to save rollback config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: err.Error()})
		return
	}

	// 同步到 ZooKeeper
	var cfg models.YafConfig
	json.Unmarshal([]byte(record.ConfigJSON), &cfg)

	var zkPath string
	switch scope {
	case models.ScopeGlobal:
		zkPath = zk.GetGlobalConfigPath()
	case models.ScopeCluster:
		zkPath = zk.GetClusterConfigPath(req.ClusterName)
	case models.ScopeNode:
		zkPath = zk.GetNodeConfigPath(req.ClusterName, req.NodeID)
	}

	if err := h.zkClient.SetConfig(zkPath, cfg); err != nil {
		h.logger.Error("failed to sync rollback config to zk", zap.Error(err))
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "rollback success",
		Data:    map[string]int{"new_version": newRecord.Version},
	})
}

