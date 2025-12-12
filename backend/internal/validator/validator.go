package validator

import (
	"fmt"
	"net"
	"strings"

	"github.com/yf-web/backend/internal/models"
)

// ConfigValidator 配置验证器
type ConfigValidator struct{}

// NewConfigValidator 创建验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate 验证配置
func (v *ConfigValidator) Validate(cfg *models.YafConfig) error {
	if err := v.validateCapture(&cfg.Capture); err != nil {
		return fmt.Errorf("capture config error: %w", err)
	}
	if err := v.validateFilter(&cfg.Filter); err != nil {
		return fmt.Errorf("filter config error: %w", err)
	}
	if err := v.validateOutput(&cfg.Output); err != nil {
		return fmt.Errorf("output config error: %w", err)
	}
	return nil
}

// validateCapture 验证采集配置
func (v *ConfigValidator) validateCapture(cfg *models.CaptureConfig) error {
	// 验证 IPFIX 端口
	if cfg.IPFIXPort < 0 || cfg.IPFIXPort > 65535 {
		return fmt.Errorf("ipfix_port must be between 0 and 65535, got %d", cfg.IPFIXPort)
	}
	// 验证空闲超时
	if cfg.IdleTimeout < 0 || cfg.IdleTimeout > 3600 {
		return fmt.Errorf("idle_timeout must be between 0 and 3600 seconds, got %d", cfg.IdleTimeout)
	}
	// 验证活跃超时
	if cfg.ActiveTimeout < 0 || cfg.ActiveTimeout > 3600 {
		return fmt.Errorf("active_timeout must be between 0 and 3600 seconds, got %d", cfg.ActiveTimeout)
	}
	// 验证统计间隔
	if cfg.StatsInterval < 0 || cfg.StatsInterval > 3600 {
		return fmt.Errorf("stats_interval must be between 0 and 3600 seconds, got %d", cfg.StatsInterval)
	}
	// 验证最大载荷
	if cfg.MaxPayload < 0 {
		return fmt.Errorf("max_payload must be non-negative, got %d", cfg.MaxPayload)
	}
	if cfg.MaxPayload > 65535 {
		return fmt.Errorf("max_payload must be at most 65535, got %d", cfg.MaxPayload)
	}
	return nil
}

// validateFilter 验证过滤配置
func (v *ConfigValidator) validateFilter(cfg *models.FilterConfig) error {
	// 验证 IP 白名单
	for _, cidr := range cfg.IPWhitelist {
		if err := v.validateCIDR(cidr); err != nil {
			return fmt.Errorf("invalid ip_whitelist entry '%s': %w", cidr, err)
		}
	}
	// 验证 IP 黑名单
	for _, cidr := range cfg.IPBlacklist {
		if err := v.validateCIDR(cidr); err != nil {
			return fmt.Errorf("invalid ip_blacklist entry '%s': %w", cidr, err)
		}
	}
	// 验证源端口
	for _, port := range cfg.SrcPorts {
		if err := v.validatePort(port); err != nil {
			return fmt.Errorf("invalid src_port %d: %w", port, err)
		}
	}
	// 验证目的端口
	for _, port := range cfg.DstPorts {
		if err := v.validatePort(port); err != nil {
			return fmt.Errorf("invalid dst_port %d: %w", port, err)
		}
	}
	return nil
}

// validateOutput 验证输出配置
func (v *ConfigValidator) validateOutput(cfg *models.OutputConfig) error {
	if len(cfg.Fields) == 0 {
		return fmt.Errorf("at least one output field is required")
	}
	supportedSet := make(map[string]bool)
	for _, f := range models.SupportedFields {
		supportedSet[f] = true
	}
	for _, field := range cfg.Fields {
		if !supportedSet[field] {
			return fmt.Errorf("unsupported output field '%s'", field)
		}
	}
	return nil
}

// validateCIDR 验证 CIDR 格式
func (v *ConfigValidator) validateCIDR(cidr string) error {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return fmt.Errorf("empty CIDR")
	}
	// 支持单个 IP 或 CIDR
	if !strings.Contains(cidr, "/") {
		if net.ParseIP(cidr) == nil {
			return fmt.Errorf("invalid IP address")
		}
		return nil
	}
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR format: %w", err)
	}
	return nil
}

// validatePort 验证端口号
func (v *ConfigValidator) validatePort(port int) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	return nil
}

// ValidateClusterName 验证集群名称
func (v *ConfigValidator) ValidateClusterName(name string) error {
	if name == "" {
		return fmt.Errorf("cluster name is required")
	}
	if len(name) > 128 {
		return fmt.Errorf("cluster name too long (max 128 characters)")
	}
	// 只允许字母、数字、下划线、中划线
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return fmt.Errorf("cluster name contains invalid character: %c", c)
		}
	}
	return nil
}

// ValidateNodeID 验证节点 ID
func (v *ConfigValidator) ValidateNodeID(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	if len(nodeID) > 128 {
		return fmt.Errorf("node ID too long (max 128 characters)")
	}
	// 只允许字母、数字、下划线、中划线、点
	for _, c := range nodeID {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.') {
			return fmt.Errorf("node ID contains invalid character: %c", c)
		}
	}
	return nil
}

