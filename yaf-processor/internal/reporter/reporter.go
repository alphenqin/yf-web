package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// StatusData 状态上报数据结构
type StatusData struct {
	// 当前值
	CurRcvPkts  int64 `json:"curRcvPkts"`  // 当前收到的包数
	CurRcvBytes int64 `json:"curRcvBytes"` // 当前收到的字节数
	CurPkts     int64 `json:"curPkts"`     // 当前处理的包数
	CurBytes    int64 `json:"curBytes"`    // 当前处理的字节数

	// 当前平均值
	CurAvgRcvPps int64 `json:"curAvgRcvPps"` // 当前每秒收到的包数
	CurAvgRcvBps int64 `json:"curAvgRcvBps"` // 当前每秒收到的字节数
	CurAvgPps    int64 `json:"curAvgPps"`    // 当前每秒处理的包数
	CurAvgBps    int64 `json:"curAvgBps"`    // 当前每秒处理的字节数

	// 容器信息
	UUID    string `json:"uuid"`    // 容器的主机名
	RunSecs int64  `json:"runSecs"` // 程序运行时间(秒)

	// 总计值
	TotalRcvPkts  int64 `json:"totalRcvPkts"`  // 总共收到的包数
	TotalRcvBytes int64 `json:"totalRcvBytes"` // 总共收到的字节数
	TotalPkts     int64 `json:"totalPkts"`     // 总共处理的包数
	TotalBytes    int64 `json:"totalBytes"`    // 总共处理的字节数

	// 总计平均值
	TotalAvgRcvPps int64 `json:"totalAvgRcvPps"` // 总共每秒收到的包数
	TotalAvgRcvBps int64 `json:"totalAvgRcvBps"` // 总共每秒收到的字节数
	TotalAvgPps    int64 `json:"totalAvgPps"`    // 总共每秒处理的包数
	TotalAvgBps    int64 `json:"totalAvgBps"`    // 总共每秒处理的字节数
}

// Reporter 状态上报器
type Reporter struct {
	reportURL         string
	reportIntervalSec int
	uuid              string
	startTime         time.Time
	lastReportTime    time.Time
	lastRcvPkts       int64
	lastRcvBytes      int64
	lastPkts          int64
	lastBytes         int64
	totalRcvPkts      int64
	totalRcvBytes     int64
	totalPkts         int64
	totalBytes        int64
	mu                sync.RWMutex
	stopChan          chan struct{}
	doneChan          chan struct{}
	httpClient        *http.Client
}

// NewReporter 创建新的状态上报器
func NewReporter(reportURL string, reportIntervalSec int, uuid string) *Reporter {
	if uuid == "" {
		// 尝试从环境变量获取主机名
		uuid = os.Getenv("HOSTNAME")
		if uuid == "" {
			hostname, err := os.Hostname()
			if err == nil {
				uuid = hostname
			} else {
				uuid = "unknown"
			}
		}
	}

	return &Reporter{
		reportURL:         reportURL,
		reportIntervalSec: reportIntervalSec,
		uuid:              uuid,
		startTime:         time.Now(),
		lastReportTime:    time.Now(),
		stopChan:          make(chan struct{}),
		doneChan:          make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start 启动状态上报器
func (r *Reporter) Start() {
	go r.run()
}

// Stop 停止状态上报器
func (r *Reporter) Stop() {
	close(r.stopChan)
	<-r.doneChan
}

// AddReceived 增加接收的包数和字节数
func (r *Reporter) AddReceived(pkts int64, bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalRcvPkts += pkts
	r.totalRcvBytes += bytes
}

// AddProcessed 增加处理的包数和字节数
func (r *Reporter) AddProcessed(pkts int64, bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalPkts += pkts
	r.totalBytes += bytes
}

// run 主循环：定时上报状态
func (r *Reporter) run() {
	defer close(r.doneChan)

	if r.reportURL == "" {
		log.Printf("[INFO] 状态上报 URL 未配置，跳过状态上报")
		return
	}

	ticker := time.NewTicker(time.Duration(r.reportIntervalSec) * time.Second)
	defer ticker.Stop()

	// 立即执行一次
	r.report()

	for {
		select {
		case <-ticker.C:
			r.report()
		case <-r.stopChan:
			// 退出前最后上报一次
			r.report()
			return
		}
	}
}

// report 执行一次状态上报
func (r *Reporter) report() {
	r.mu.Lock()
	now := time.Now()
	runSecs := int64(now.Sub(r.startTime).Seconds())

	// 计算当前周期内的增量
	curRcvPkts := r.totalRcvPkts - r.lastRcvPkts
	curRcvBytes := r.totalRcvBytes - r.lastRcvBytes
	curPkts := r.totalPkts - r.lastPkts
	curBytes := r.totalBytes - r.lastBytes

	// 计算时间间隔（秒）
	intervalSecs := int64(now.Sub(r.lastReportTime).Seconds())
	if intervalSecs == 0 {
		intervalSecs = 1 // 避免除零
	}

	// 计算当前平均值（每秒）
	curAvgRcvPps := curRcvPkts / intervalSecs
	curAvgRcvBps := curRcvBytes / intervalSecs
	curAvgPps := curPkts / intervalSecs
	curAvgBps := curBytes / intervalSecs

	// 计算总计平均值（每秒）
	totalAvgRcvPps := int64(0)
	totalAvgRcvBps := int64(0)
	totalAvgPps := int64(0)
	totalAvgBps := int64(0)
	if runSecs > 0 {
		totalAvgRcvPps = r.totalRcvPkts / runSecs
		totalAvgRcvBps = r.totalRcvBytes / runSecs
		totalAvgPps = r.totalPkts / runSecs
		totalAvgBps = r.totalBytes / runSecs
	}

	// 构建上报数据
	statusData := StatusData{
		CurRcvPkts:     curRcvPkts,
		CurRcvBytes:    curRcvBytes,
		CurPkts:        curPkts,
		CurBytes:       curBytes,
		CurAvgRcvPps:   curAvgRcvPps,
		CurAvgRcvBps:   curAvgRcvBps,
		CurAvgPps:      curAvgPps,
		CurAvgBps:      curAvgBps,
		UUID:           r.uuid,
		RunSecs:        runSecs,
		TotalRcvPkts:   r.totalRcvPkts,
		TotalRcvBytes:  r.totalRcvBytes,
		TotalPkts:      r.totalPkts,
		TotalBytes:     r.totalBytes,
		TotalAvgRcvPps: totalAvgRcvPps,
		TotalAvgRcvBps: totalAvgRcvBps,
		TotalAvgPps:    totalAvgPps,
		TotalAvgBps:    totalAvgBps,
	}

	// 更新上次上报的值
	r.lastReportTime = now
	r.lastRcvPkts = r.totalRcvPkts
	r.lastRcvBytes = r.totalRcvBytes
	r.lastPkts = r.totalPkts
	r.lastBytes = r.totalBytes
	r.mu.Unlock()

	// 发送 HTTP POST 请求
	if err := r.sendReport(statusData); err != nil {
		log.Printf("[ERROR] 状态上报失败: %v", err)
	} else {
		log.Printf("[INFO] 状态上报成功: 运行时间=%ds, 总接收=%d包/%d字节, 总处理=%d包/%d字节",
			runSecs, r.totalRcvPkts, r.totalRcvBytes, r.totalPkts, r.totalBytes)
	}
}

// sendReport 发送状态上报请求
func (r *Reporter) sendReport(data StatusData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	req, err := http.NewRequest("POST", r.reportURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}
