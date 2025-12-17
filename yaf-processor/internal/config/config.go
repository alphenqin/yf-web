package config

import (
	"fmt"
	"os"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// containsProcessorConfig 检查字符串中是否包含 processor 配置块
func containsProcessorConfig(content string) bool {
	// 转换为小写进行匹配，更宽松的检测
	contentLower := strings.ToLower(content)
	// 检查是否包含 processor 关键字
	if strings.Contains(contentLower, "processor") {
		return true
	}
	return false
}

// ProcessorConfig 包含 processor 的所有配置项
type ProcessorConfig struct {
	// 文件滚动配置
	RotateIntervalSec int
	RotateSizeMB      int
	FilePrefix        string
	Timezone          string // 时区，例如 "Asia/Shanghai" 或 "UTC+8"，默认为 "Asia/Shanghai"
	// 状态上报配置
	StatusReportURL         string // 状态上报 URL，例如 "http://example.com/api/uploadStatus"
	StatusReportIntervalSec int    // 状态上报间隔（秒），默认 60
	UUID                    string // 容器主机名，如果为空则从环境变量 HOSTNAME 获取
	// 输出目标配置（预留，未来用于配置 Kafka/MQ 等）
	OutputType   string                 // 输出类型，例如 "kafka", "mq", "file" 等
	OutputConfig map[string]interface{} // 输出目标的具体配置（JSON 格式）
}

// LoadConfig 从 yaf.init 文件中解析 processor 配置块
func LoadConfig(configPath string) (*ProcessorConfig, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("配置文件不存在: %w", err)
	}

	// 使用 gopher-lua 解析 Lua 配置文件
	L := lua.NewState()
	defer L.Close()

	// 读取文件内容用于调试
	fileContent, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 执行 Lua 文件
	if err := L.DoFile(configPath); err != nil {
		return nil, fmt.Errorf("执行 Lua 文件失败: %w", err)
	}

	// 获取 processor 配置
	tbl := L.GetGlobal("processor")

	// 如果找不到 processor 配置，返回错误
	if tbl == nil || tbl == lua.LNil {
		var foundGlobals []string
		testGlobals := []string{"input", "output", "applabel", "filter", "processor"}
		for _, name := range testGlobals {
			if val := L.GetGlobal(name); val != nil && val != lua.LNil {
				foundGlobals = append(foundGlobals, name)
			}
		}

		// 检查配置文件中是否包含 processor 关键字
		contentStr := string(fileContent)
		hasProcessorConfig := containsProcessorConfig(contentStr)

		errMsg := fmt.Sprintf("未找到 processor 配置块。检测到的全局变量: %v", foundGlobals)
		if !hasProcessorConfig {
			errMsg += " (配置文件中未找到 processor 关键字，请检查配置文件)"
		}
		return nil, fmt.Errorf(errMsg)
	}

	flowTbl, ok := tbl.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("processor 配置块格式错误: 期望表类型，实际类型: %T", tbl)
	}

	// 从 flowTbl 中读取配置项
	cfg := &ProcessorConfig{
		OutputConfig: make(map[string]interface{}),
	}

	// 读取文件滚动配置
	if val := flowTbl.RawGetString("file_prefix"); val != nil && val != lua.LNil {
		cfg.FilePrefix = val.String()
	}
	if val := flowTbl.RawGetString("rotate_interval_sec"); val != nil && val != lua.LNil {
		if num, ok := val.(lua.LNumber); ok {
			cfg.RotateIntervalSec = int(num)
		}
	}
	if val := flowTbl.RawGetString("rotate_size_mb"); val != nil && val != lua.LNil {
		if num, ok := val.(lua.LNumber); ok {
			cfg.RotateSizeMB = int(num)
		}
	}
	if val := flowTbl.RawGetString("timezone"); val != nil && val != lua.LNil {
		cfg.Timezone = val.String()
	}

	// 读取状态上报配置
	if val := flowTbl.RawGetString("status_report_url"); val != nil && val != lua.LNil {
		cfg.StatusReportURL = val.String()
	}
	if val := flowTbl.RawGetString("status_report_interval_sec"); val != nil && val != lua.LNil {
		if num, ok := val.(lua.LNumber); ok {
			cfg.StatusReportIntervalSec = int(num)
		}
	}
	if val := flowTbl.RawGetString("uuid"); val != nil && val != lua.LNil {
		cfg.UUID = val.String()
	}

	// 读取输出类型配置
	if val := flowTbl.RawGetString("output_type"); val != nil && val != lua.LNil {
		cfg.OutputType = val.String()
	}

	// 读取输出配置（output_config 是一个 Lua 表，需要解析）
	if val := flowTbl.RawGetString("output_config"); val != nil && val != lua.LNil {
		if outputTbl, ok := val.(*lua.LTable); ok {
			// 解析 output_config 表
			outputTbl.ForEach(func(key, value lua.LValue) {
				keyStr := key.String()
				switch v := value.(type) {
				case lua.LString:
					cfg.OutputConfig[keyStr] = v.String()
				case lua.LNumber:
					cfg.OutputConfig[keyStr] = float64(v)
				case lua.LBool:
					cfg.OutputConfig[keyStr] = bool(v)
				case *lua.LTable:
					// 如果是表，转换为字符串数组（简化处理）
					var arr []string
					v.ForEach(func(k, v lua.LValue) {
						arr = append(arr, v.String())
					})
					cfg.OutputConfig[keyStr] = arr
				}
			})
		}
	}

	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// validateConfig 验证配置的有效性
func validateConfig(cfg *ProcessorConfig) error {
	if cfg.RotateIntervalSec < 1 {
		cfg.RotateIntervalSec = 60 // 默认 60 秒
	}
	if cfg.RotateSizeMB < 1 {
		cfg.RotateSizeMB = 100 // 默认 100 MB
	}
	if cfg.FilePrefix == "" {
		cfg.FilePrefix = "flows_"
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "Asia/Shanghai" // 默认东八区
	}
	if cfg.StatusReportIntervalSec < 1 {
		cfg.StatusReportIntervalSec = 60 // 默认 60 秒
	}
	if cfg.OutputType == "" {
		cfg.OutputType = "file" // 默认输出到文件
	}

	return nil
}

// EnsureDataDir 确保数据目录存在
func EnsureDataDir(dataDir string) error {
	info, err := os.Stat(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return fmt.Errorf("创建数据目录失败: %w", err)
			}
			return nil
		}
		return fmt.Errorf("检查数据目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("数据目录路径不是目录: %s", dataDir)
	}
	return nil
}
