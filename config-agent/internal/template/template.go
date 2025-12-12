package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

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

-- ========= flow2ftp 扩展配置 =========
flow2ftp = {
  ftp_host = "{{ .FTPHost }}",
  ftp_port = {{ .FTPPort }},
  ftp_user = "{{ .FTPUser }}",
  ftp_pass = "{{ .FTPPass }}",
  ftp_dir  = "{{ .FTPDir }}",
  rotate_interval_sec = 60,
  rotate_size_mb      = 100,
  file_prefix         = "flows_",
  upload_interval_sec = 60,
  timezone = "Asia/Shanghai",
  
  -- 输出字段列表
  output_fields = {
{{- range $i, $field := .OutputFields }}
    "{{ $field }}",
{{- end }}
  }
}
`

// TemplateData 模板数据
type TemplateData struct {
	GeneratedAt   string
	Cluster       string
	NodeID        string
	Interface     string
	IPFIXPort     int
	IdleTimeout   int
	ActiveTimeout int
	StatsInterval int
	BPFFilter     string
	AppLabel      string
	DPI           string
	MaxPayload    int
	FTPHost       string
	FTPPort       int
	FTPUser       string
	FTPPass       string
	FTPDir        string
	OutputFields  []string
}

// Generator 配置文件生成器
type Generator struct {
	configPath string
	interface_ string
	ipfixPort  string
	ftpHost    string
	ftpPort    int
	ftpUser    string
	ftpPass    string
	ftpDir     string
	cluster    string
	nodeID     string
	logger     *zap.Logger
	tmpl       *template.Template
}

// NewGenerator 创建生成器
func NewGenerator(
	configPath, interface_, ipfixPort string,
	ftpHost string, ftpPort int, ftpUser, ftpPass, ftpDir string,
	cluster, nodeID string,
	logger *zap.Logger,
) (*Generator, error) {
	tmpl, err := template.New("yaf.init").Parse(YafInitTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Generator{
		configPath: configPath,
		interface_: interface_,
		ipfixPort:  ipfixPort,
		ftpHost:    ftpHost,
		ftpPort:    ftpPort,
		ftpUser:    ftpUser,
		ftpPass:    ftpPass,
		ftpDir:     ftpDir,
		cluster:    cluster,
		nodeID:     nodeID,
		logger:     logger,
		tmpl:       tmpl,
	}, nil
}

// Generate 生成配置文件
func (g *Generator) Generate(cfg *config.YafConfig) error {
	// 使用配置中的值，如果为空则使用环境变量的默认值
	iface := cfg.Capture.Interface
	if iface == "" {
		iface = g.interface_
	}
	ipfixPort := cfg.Capture.IPFIXPort
	if ipfixPort == 0 {
		ipfixPort = 18000
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

	// 准备模板数据
	data := TemplateData{
		GeneratedAt:   "auto-generated",
		Cluster:       g.cluster,
		NodeID:        g.nodeID,
		Interface:     iface,
		IPFIXPort:     ipfixPort,
		IdleTimeout:   idleTimeout,
		ActiveTimeout: activeTimeout,
		StatsInterval: statsInterval,
		BPFFilter:     g.buildBPFFilter(cfg),
		AppLabel:      boolToLua(cfg.Capture.EnableAppLabel),
		DPI:           boolToLua(cfg.Capture.EnableDPI),
		MaxPayload:    cfg.Capture.MaxPayload,
		FTPHost:       g.ftpHost,
		FTPPort:       g.ftpPort,
		FTPUser:       g.ftpUser,
		FTPPass:       g.ftpPass,
		FTPDir:        g.ftpDir,
		OutputFields:  outputFields,
	}

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

	g.logger.Info("config file generated", zap.String("path", g.configPath))
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

