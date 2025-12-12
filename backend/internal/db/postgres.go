package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/yf-web/backend/internal/models"
	"go.uber.org/zap"
)

// PostgresDB PostgreSQL 数据库封装
type PostgresDB struct {
	db     *sql.DB
	logger *zap.Logger
}

// Config 数据库配置
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewPostgresDB 创建数据库连接
func NewPostgresDB(cfg Config, logger *zap.Logger) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pdb := &PostgresDB{db: db, logger: logger}

	// 初始化表结构
	if err := pdb.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return pdb, nil
}

// initSchema 初始化表结构
func (p *PostgresDB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS yaf_config (
		id BIGSERIAL PRIMARY KEY,
		scope VARCHAR(16) NOT NULL,
		cluster_name VARCHAR(128),
		node_id VARCHAR(128),
		version INT NOT NULL DEFAULT 1,
		config_json JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		created_by VARCHAR(128) NOT NULL,
		UNIQUE(scope, cluster_name, node_id, version)
	);
	
	CREATE INDEX IF NOT EXISTS idx_yaf_config_scope ON yaf_config(scope);
	CREATE INDEX IF NOT EXISTS idx_yaf_config_cluster ON yaf_config(cluster_name);
	CREATE INDEX IF NOT EXISTS idx_yaf_config_node ON yaf_config(node_id);
	CREATE INDEX IF NOT EXISTS idx_yaf_config_created_at ON yaf_config(created_at);

	-- 用户表
	CREATE TABLE IF NOT EXISTS yaf_users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(64) NOT NULL UNIQUE,
		password VARCHAR(128) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- 系统设置表
	CREATE TABLE IF NOT EXISTS yaf_settings (
		key VARCHAR(64) PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`

	_, err := p.db.Exec(schema)
	if err != nil {
		return err
	}

	// 初始化默认管理员账号
	return p.initDefaultUser()
}

// initDefaultUser 初始化默认用户
func (p *PostgresDB) initDefaultUser() error {
	var count int
	err := p.db.QueryRow("SELECT COUNT(*) FROM yaf_users").Scan(&count)
	if err != nil {
		return err
	}

	// 如果没有用户，创建默认管理员
	if count == 0 {
		_, err = p.db.Exec(
			"INSERT INTO yaf_users (username, password) VALUES ($1, $2)",
			"admin", "admin",
		)
		if err != nil {
			return fmt.Errorf("failed to create default user: %w", err)
		}
		p.logger.Info("created default admin user (admin/admin)")
	}
	return nil
}

// Close 关闭数据库连接
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// SaveConfig 保存配置（新版本）
func (p *PostgresDB) SaveConfig(record *models.ConfigRecord) error {
	// 获取最新版本号
	var maxVersion int
	err := p.db.QueryRow(`
		SELECT COALESCE(MAX(version), 0) FROM yaf_config 
		WHERE scope = $1 AND COALESCE(cluster_name, '') = $2 AND COALESCE(node_id, '') = $3
	`, record.Scope, record.ClusterName, record.NodeID).Scan(&maxVersion)
	if err != nil {
		return fmt.Errorf("failed to get max version: %w", err)
	}

	record.Version = maxVersion + 1
	record.CreatedAt = time.Now()

	_, err = p.db.Exec(`
		INSERT INTO yaf_config (scope, cluster_name, node_id, version, config_json, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, record.Scope, record.ClusterName, record.NodeID, record.Version, record.ConfigJSON, record.CreatedAt, record.CreatedBy)

	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	p.logger.Info("config saved to database",
		zap.String("scope", string(record.Scope)),
		zap.String("cluster", record.ClusterName),
		zap.String("node", record.NodeID),
		zap.Int("version", record.Version),
	)
	return nil
}

// GetLatestConfig 获取最新配置
func (p *PostgresDB) GetLatestConfig(scope models.ConfigScope, clusterName, nodeID string) (*models.ConfigRecord, error) {
	record := &models.ConfigRecord{}
	err := p.db.QueryRow(`
		SELECT id, scope, cluster_name, node_id, version, config_json, created_at, created_by
		FROM yaf_config
		WHERE scope = $1 AND COALESCE(cluster_name, '') = $2 AND COALESCE(node_id, '') = $3
		ORDER BY version DESC
		LIMIT 1
	`, scope, clusterName, nodeID).Scan(
		&record.ID, &record.Scope, &record.ClusterName, &record.NodeID,
		&record.Version, &record.ConfigJSON, &record.CreatedAt, &record.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return record, nil
}

// GetConfigHistory 获取配置历史
func (p *PostgresDB) GetConfigHistory(scope models.ConfigScope, clusterName, nodeID string, limit int) ([]*models.ConfigRecord, error) {
	rows, err := p.db.Query(`
		SELECT id, scope, cluster_name, node_id, version, config_json, created_at, created_by
		FROM yaf_config
		WHERE scope = $1 AND COALESCE(cluster_name, '') = $2 AND COALESCE(node_id, '') = $3
		ORDER BY version DESC
		LIMIT $4
	`, scope, clusterName, nodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query config history: %w", err)
	}
	defer rows.Close()

	var records []*models.ConfigRecord
	for rows.Next() {
		record := &models.ConfigRecord{}
		if err := rows.Scan(
			&record.ID, &record.Scope, &record.ClusterName, &record.NodeID,
			&record.Version, &record.ConfigJSON, &record.CreatedAt, &record.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}

// GetConfigByVersion 获取指定版本配置
func (p *PostgresDB) GetConfigByVersion(scope models.ConfigScope, clusterName, nodeID string, version int) (*models.ConfigRecord, error) {
	record := &models.ConfigRecord{}
	err := p.db.QueryRow(`
		SELECT id, scope, cluster_name, node_id, version, config_json, created_at, created_by
		FROM yaf_config
		WHERE scope = $1 AND COALESCE(cluster_name, '') = $2 AND COALESCE(node_id, '') = $3 AND version = $4
	`, scope, clusterName, nodeID, version).Scan(
		&record.ID, &record.Scope, &record.ClusterName, &record.NodeID,
		&record.Version, &record.ConfigJSON, &record.CreatedAt, &record.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return record, nil
}

// ListClusters 列出数据库中的所有集群
func (p *PostgresDB) ListClusters() ([]string, error) {
	rows, err := p.db.Query(`
		SELECT DISTINCT cluster_name FROM yaf_config 
		WHERE cluster_name IS NOT NULL AND cluster_name != ''
		ORDER BY cluster_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer rows.Close()

	var clusters []string
	for rows.Next() {
		var cluster string
		if err := rows.Scan(&cluster); err != nil {
			return nil, fmt.Errorf("failed to scan cluster: %w", err)
		}
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

// ListNodes 列出集群下的所有节点
func (p *PostgresDB) ListNodes(clusterName string) ([]string, error) {
	rows, err := p.db.Query(`
		SELECT DISTINCT node_id FROM yaf_config 
		WHERE cluster_name = $1 AND node_id IS NOT NULL AND node_id != ''
		ORDER BY node_id
	`, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []string
	for rows.Next() {
		var node string
		if err := rows.Scan(&node); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// ValidateUser 验证用户登录
func (p *PostgresDB) ValidateUser(username, password string) (bool, error) {
	var storedPassword string
	err := p.db.QueryRow(
		"SELECT password FROM yaf_users WHERE username = $1",
		username,
	).Scan(&storedPassword)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to query user: %w", err)
	}

	return storedPassword == password, nil
}

// GetSetting 获取系统设置
func (p *PostgresDB) GetSetting(key string) (string, error) {
	var value string
	err := p.db.QueryRow(
		"SELECT value FROM yaf_settings WHERE key = $1",
		key,
	).Scan(&value)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}

// SetSetting 保存系统设置
func (p *PostgresDB) SetSetting(key, value string) error {
	_, err := p.db.Exec(`
		INSERT INTO yaf_settings (key, value, updated_at) 
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`, key, value)
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}
	return nil
}

// GetAllSettings 获取所有系统设置
func (p *PostgresDB) GetAllSettings() (map[string]string, error) {
	rows, err := p.db.Query("SELECT key, value FROM yaf_settings")
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings[key] = value
	}
	return settings, nil
}

