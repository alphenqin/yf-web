package config

// CaptureConfig 采集配置
type CaptureConfig struct {
	Interface      string `json:"interface"`        // 网卡名称，如 eth0
	IPFIXPort      int    `json:"ipfix_port"`       // IPFIX 输出端口，如 18000
	IdleTimeout    int    `json:"idle_timeout"`     // 空闲超时（秒）
	ActiveTimeout  int    `json:"active_timeout"`   // 活跃超时（秒）
	StatsInterval  int    `json:"stats_interval"`   // 统计输出间隔（秒）
	EnableAppLabel bool   `json:"enable_applabel"`  // 启用应用识别
	EnableDPI      bool   `json:"enable_dpi"`       // 启用 DPI
	MaxPayload     int    `json:"max_payload"`      // 最大载荷字节数
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

// YafConfig YAF 完整配置
type YafConfig struct {
	Capture CaptureConfig `json:"capture"`
	Filter  FilterConfig  `json:"filter"`
	Output  OutputConfig  `json:"output"`
}

// MergeConfig 合并配置，后者覆盖前者
func MergeConfig(base, overlay *YafConfig) *YafConfig {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	merged := &YafConfig{}

	// Capture 配置合并
	merged.Capture.Interface = base.Capture.Interface
	if overlay.Capture.Interface != "" {
		merged.Capture.Interface = overlay.Capture.Interface
	}
	merged.Capture.IPFIXPort = base.Capture.IPFIXPort
	if overlay.Capture.IPFIXPort > 0 {
		merged.Capture.IPFIXPort = overlay.Capture.IPFIXPort
	}
	merged.Capture.IdleTimeout = base.Capture.IdleTimeout
	if overlay.Capture.IdleTimeout > 0 {
		merged.Capture.IdleTimeout = overlay.Capture.IdleTimeout
	}
	merged.Capture.ActiveTimeout = base.Capture.ActiveTimeout
	if overlay.Capture.ActiveTimeout > 0 {
		merged.Capture.ActiveTimeout = overlay.Capture.ActiveTimeout
	}
	merged.Capture.StatsInterval = base.Capture.StatsInterval
	if overlay.Capture.StatsInterval > 0 {
		merged.Capture.StatsInterval = overlay.Capture.StatsInterval
	}
	merged.Capture.EnableAppLabel = base.Capture.EnableAppLabel || overlay.Capture.EnableAppLabel
	merged.Capture.EnableDPI = base.Capture.EnableDPI || overlay.Capture.EnableDPI
	merged.Capture.MaxPayload = base.Capture.MaxPayload
	if overlay.Capture.MaxPayload > 0 {
		merged.Capture.MaxPayload = overlay.Capture.MaxPayload
	}

	// Filter 配置合并（覆盖策略）
	merged.Filter.IPWhitelist = base.Filter.IPWhitelist
	if len(overlay.Filter.IPWhitelist) > 0 {
		merged.Filter.IPWhitelist = overlay.Filter.IPWhitelist
	}
	merged.Filter.IPBlacklist = base.Filter.IPBlacklist
	if len(overlay.Filter.IPBlacklist) > 0 {
		merged.Filter.IPBlacklist = overlay.Filter.IPBlacklist
	}
	merged.Filter.SrcPorts = base.Filter.SrcPorts
	if len(overlay.Filter.SrcPorts) > 0 {
		merged.Filter.SrcPorts = overlay.Filter.SrcPorts
	}
	merged.Filter.DstPorts = base.Filter.DstPorts
	if len(overlay.Filter.DstPorts) > 0 {
		merged.Filter.DstPorts = overlay.Filter.DstPorts
	}
	merged.Filter.BPFFilter = base.Filter.BPFFilter
	if overlay.Filter.BPFFilter != "" {
		merged.Filter.BPFFilter = overlay.Filter.BPFFilter
	}

	// Output 配置合并（覆盖策略）
	merged.Output.Fields = base.Output.Fields
	if len(overlay.Output.Fields) > 0 {
		merged.Output.Fields = overlay.Output.Fields
	}

	return merged
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
	}
}

