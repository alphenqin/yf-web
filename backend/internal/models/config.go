package models

import "time"

// CaptureConfig 采集配置
type CaptureConfig struct {
	Interface      string `json:"interface"`       // 网卡名称，如 eth0
	IPFIXPort      int    `json:"ipfix_port"`      // IPFIX 输出端口，如 18000
	IdleTimeout    int    `json:"idle_timeout"`    // 空闲超时（秒）
	ActiveTimeout  int    `json:"active_timeout"`  // 活跃超时（秒）
	StatsInterval  int    `json:"stats_interval"`  // 统计输出间隔（秒）
	EnableAppLabel bool   `json:"enable_applabel"` // 启用应用识别
	EnableDPI      bool   `json:"enable_dpi"`      // 启用 DPI
	MaxPayload     int    `json:"max_payload"`     // 最大载荷字节数
}

// FilterConfig 过滤配置
type FilterConfig struct {
	IPWhitelist []string `json:"ip_whitelist"`
	IPBlacklist []string `json:"ip_blacklist"`
	SrcPorts    []int    `json:"src_ports"`
	DstPorts    []int    `json:"dst_ports"`
	BPFFilter   string   `json:"bpf_filter"`
}

// OutputConfig 输出配置
type OutputConfig struct {
	Fields []string `json:"fields"`
}

// StatusReportConfig 状态上报配置
type StatusReportConfig struct {
	StatusReportURL         string `json:"status_report_url"`          // 状态上报 URL，例如 "http://example.com/api/uploadStatus"
	StatusReportIntervalSec int    `json:"status_report_interval_sec"` // 状态上报间隔（秒），默认 60
	UUID                    string `json:"uuid"`                       // 容器主机名，如果为空则从环境变量 HOSTNAME 获取
}

// YafConfig YAF 完整配置
type YafConfig struct {
	Capture      CaptureConfig      `json:"capture"`
	Filter       FilterConfig       `json:"filter"`
	Output       OutputConfig       `json:"output"`
	StatusReport StatusReportConfig `json:"status_report"`
}

// ConfigScope 配置作用范围
type ConfigScope string

const (
	ScopeGlobal  ConfigScope = "global"
	ScopeCluster ConfigScope = "cluster"
	ScopeNode    ConfigScope = "node"
)

// ConfigRecord 数据库配置记录
type ConfigRecord struct {
	ID          int64       `json:"id" db:"id"`
	Scope       ConfigScope `json:"scope" db:"scope"`
	ClusterName string      `json:"cluster_name,omitempty" db:"cluster_name"`
	NodeID      string      `json:"node_id,omitempty" db:"node_id"`
	Version     int         `json:"version" db:"version"`
	ConfigJSON  string      `json:"config_json" db:"config_json"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	CreatedBy   string      `json:"created_by" db:"created_by"`
}

// SupportedFields YAF 支持的所有输出字段
var SupportedFields = []string{
	"flowStartMilliseconds",
	"flowEndMilliseconds",
	"sourceIPv4Address",
	"destinationIPv4Address",
	"sourceTransportPort",
	"destinationTransportPort",
	"protocolIdentifier",
	"silkAppLabel",
	"octetTotalCount",
	"packetTotalCount",
	"initialTCPFlags",
	"ipClassOfService",
	"ingressInterface",
	"egressInterface",
}

// FieldLabels 字段中文名称
var FieldLabels = map[string]string{
	"flowStartMilliseconds":    "流开始时间",
	"flowEndMilliseconds":      "流结束时间",
	"sourceIPv4Address":        "源IP地址",
	"destinationIPv4Address":   "目的IP地址",
	"sourceTransportPort":      "源端口",
	"destinationTransportPort": "目的端口",
	"protocolIdentifier":       "协议标识",
	"silkAppLabel":             "应用标签",
	"octetTotalCount":          "字节总数",
	"packetTotalCount":         "数据包总数",
	"initialTCPFlags":          "TCP初始标志",
	"ipClassOfService":         "服务类型(ToS)",
	"ingressInterface":         "入接口",
	"egressInterface":          "出接口",
}

// DefaultConfig 默认配置
func DefaultConfig() *YafConfig {
	return &YafConfig{
		Capture: CaptureConfig{
			Interface:      "eth0",
			IPFIXPort:      18000,
			IdleTimeout:    60,
			ActiveTimeout:  60,
			StatsInterval:  300,
			EnableAppLabel: true,
			EnableDPI:      false,
			MaxPayload:     1024,
		},
		Filter: FilterConfig{
			IPWhitelist: []string{},
			IPBlacklist: []string{},
			SrcPorts:    []int{},
			DstPorts:    []int{},
			BPFFilter:   "ip and not port 22",
		},
		Output: OutputConfig{
			Fields: []string{
				"flowStartMilliseconds",
				"flowEndMilliseconds",
				"sourceIPv4Address",
				"destinationIPv4Address",
				"sourceTransportPort",
				"destinationTransportPort",
				"protocolIdentifier",
				"silkAppLabel",
			},
		},
		StatusReport: StatusReportConfig{
			StatusReportURL:         "",
			StatusReportIntervalSec: 60,
			UUID:                    "",
		},
	}
}
