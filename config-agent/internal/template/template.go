package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/yf-web/config-agent/internal/config"
	"go.uber.org/zap"
)

// YafInitTemplate yaf.init Lua 配置模板
const YafInitTemplate = `-- ========= YAF 配置 (自动生成，请勿手工修改) =========
-- 生成时间: {{ .GeneratedAt }}
-- 集群: {{ .Cluster }}
-- 节点: {{ .NodeID }}

-- 从网卡抓包
input = {
  type = "pcap",
  inf  = "{{ .Interface }}",
  export_interface = true,
}

-- 输出 IPFIX 到本机
output = {
  host     = "127.0.0.1",
  port     = "{{ .IPFIXPort }}",
  protocol = "tcp",
}

-- 解码选项
decode = {
  ip4_only = false,
  ip6_only = false,
  nofrag   = false,
}

-- 导出选项
export = {
  silk       = true,
  uniflow    = true,
  flow_stats = true,
  mac        = true,
}

-- 流超时（单位：秒）
idle_timeout   = {{ .IdleTimeout }}
active_timeout = {{ .ActiveTimeout }}

-- BPF 过滤器
filter = "{{ .BPFFilter }}"

-- 应用识别 / DPI
applabel   = {{ .AppLabel }}
dpi        = {{ .DPI }}
maxpayload = {{ .MaxPayload }}
maxexport  = {{ .MaxPayload }}
udp_payload = true
stats       = {{ .StatsInterval }}

-- ========= processor 扩展配置 =========
processor = {
  rotate_interval_sec = 60,
  rotate_size_mb      = 100,
  file_prefix         = "flows_",
  timezone = "Asia/Shanghai",
  
  -- 输出类型（file/kafka/mq等，未来扩展）
  output_type = "file",
  
  -- 输出字段列表
  output_fields = {
{{- range $i, $field := .OutputFields }}
    "{{ $field }}",
{{- end }}
  },
  
  -- 状态上报配置
  status_report_url = "{{ .StatusReportURL }}",
  status_report_interval_sec = {{ .StatusReportIntervalSec }},
{{- if .UUID }}
  uuid = "{{ .UUID }}",
{{- end }}
}
`

// TemplateData 模板数据
type TemplateData struct {
	GeneratedAt             string
	Cluster                 string
	NodeID                  string
	Interface               string
	IPFIXPort               int
	IdleTimeout             int
	ActiveTimeout           int
	StatsInterval           int
	BPFFilter               string
	AppLabel                string
	DPI                     string
	MaxPayload              int
	OutputFields            []string
	StatusReportURL         string
	StatusReportIntervalSec int
	UUID                    string
}

// Generator 配置文件生成器
type Generator struct {
	configPath string
	cluster    string
	nodeID     string
	logger     *zap.Logger
	tmpl       *template.Template
}

// NewGenerator 创建生成器
func NewGenerator(
	configPath string,
	cluster, nodeID string,
	logger *zap.Logger,
) (*Generator, error) {
	tmpl, err := template.New("yaf.init").Parse(YafInitTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Generator{
		configPath: configPath,
		cluster:    cluster,
		nodeID:     nodeID,
		logger:     logger,
		tmpl:       tmpl,
	}, nil
}

// Generate 生成配置文件
func (g *Generator) Generate(cfg *config.YafConfig) error {
	// 使用配置中的值，如果为空则使用硬编码的默认值
	iface := cfg.Capture.Interface
	if iface == "" {
		iface = "eth0" // 硬编码默认网卡名称
	}
	ipfixPort := cfg.Capture.IPFIXPort
	if ipfixPort == 0 {
		ipfixPort = 18000 // 硬编码默认 IPFIX 端口
	}
	idleTimeout := cfg.Capture.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = 60
	}
	activeTimeout := cfg.Capture.ActiveTimeout
	if activeTimeout == 0 {
		activeTimeout = 60
	}
	statsInterval := cfg.Capture.StatsInterval
	if statsInterval == 0 {
		statsInterval = 300
	}

	// 准备输出字段，如果为空则使用默认字段
	outputFields := cfg.Output.Fields
	if len(outputFields) == 0 {
		outputFields = []string{
			"flowStartMilliseconds",
			"flowEndMilliseconds",
			"sourceIPv4Address",
			"destinationIPv4Address",
			"sourceTransportPort",
			"destinationTransportPort",
			"protocolIdentifier",
			"silkAppLabel",
		}
	}

	// 准备状态上报配置
	statusReportURL := cfg.StatusReport.StatusReportURL
	statusReportIntervalSec := cfg.StatusReport.StatusReportIntervalSec
	if statusReportIntervalSec == 0 {
		statusReportIntervalSec = 60 // 默认 60 秒
	}
	uuid := cfg.StatusReport.UUID

	// 生成时间戳
	generatedAt := time.Now().Format("2006-01-02 15:04:05")

	// 准备模板数据
	data := TemplateData{
		GeneratedAt:             generatedAt,
		Cluster:                 g.cluster,
		NodeID:                  g.nodeID,
		Interface:               iface,
		IPFIXPort:               ipfixPort,
		IdleTimeout:             idleTimeout,
		ActiveTimeout:           activeTimeout,
		StatsInterval:           statsInterval,
		BPFFilter:               g.buildBPFFilter(cfg),
		AppLabel:                boolToLua(cfg.Capture.EnableAppLabel),
		DPI:                     boolToLua(cfg.Capture.EnableDPI),
		MaxPayload:              cfg.Capture.MaxPayload,
		OutputFields:            outputFields,
		StatusReportURL:         statusReportURL,
		StatusReportIntervalSec: statusReportIntervalSec,
		UUID:                    uuid,
	}

	g.logger.Info("[CONFIG_GENERATE] 开始生成配置文件",
		zap.String("config_path", g.configPath),
		zap.String("generated_at", generatedAt),
		zap.String("cluster", g.cluster),
		zap.String("node_id", g.nodeID),
	)

	// 渲染模板
	var buf bytes.Buffer
	if err := g.tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// 原子写入：先写临时文件，再重命名
	tmpPath := g.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(g.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// 原子重命名
	if err := os.Rename(tmpPath, g.configPath); err != nil {
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	g.logger.Info("[CONFIG_GENERATE] 配置文件生成成功",
		zap.String("config_path", g.configPath),
		zap.String("generated_at", generatedAt),
		zap.String("interface", iface),
		zap.Int("ipfix_port", ipfixPort),
		zap.String("bpf_filter", data.BPFFilter),
		zap.Int("output_fields_count", len(outputFields)),
	)
	return nil
}

// buildBPFFilter 构建 BPF 过滤器
func (g *Generator) buildBPFFilter(cfg *config.YafConfig) string {
	var parts []string

	// 基础过滤器
	if cfg.Filter.BPFFilter != "" {
		parts = append(parts, "("+cfg.Filter.BPFFilter+")")
	} else {
		parts = append(parts, "ip")
	}

	// IP 白名单
	if len(cfg.Filter.IPWhitelist) > 0 {
		var ipParts []string
		for _, ip := range cfg.Filter.IPWhitelist {
			ipParts = append(ipParts, fmt.Sprintf("net %s", ip))
		}
		parts = append(parts, "("+strings.Join(ipParts, " or ")+")")
	}

	// IP 黑名单
	if len(cfg.Filter.IPBlacklist) > 0 {
		for _, ip := range cfg.Filter.IPBlacklist {
			parts = append(parts, fmt.Sprintf("not net %s", ip))
		}
	}

	// 源端口
	if len(cfg.Filter.SrcPorts) > 0 {
		var portParts []string
		for _, port := range cfg.Filter.SrcPorts {
			portParts = append(portParts, "src port "+strconv.Itoa(port))
		}
		parts = append(parts, "("+strings.Join(portParts, " or ")+")")
	}

	// 目的端口
	if len(cfg.Filter.DstPorts) > 0 {
		var portParts []string
		for _, port := range cfg.Filter.DstPorts {
			portParts = append(portParts, "dst port "+strconv.Itoa(port))
		}
		parts = append(parts, "("+strings.Join(portParts, " or ")+")")
	}

	return strings.Join(parts, " and ")
}

func boolToLua(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
