package supervisor

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Controller supervisor 控制器
type Controller struct {
	logger *zap.Logger
}

// NewController 创建控制器
func NewController(logger *zap.Logger) *Controller {
	return &Controller{logger: logger}
}

// RestartYAF 重启 YAF 进程
func (c *Controller) RestartYAF() error {
	startTime := time.Now()
	c.logger.Info("[YAF_RESTART] 开始重启 YAF 进程",
		zap.Time("restart_time", startTime),
	)

	// 先检查当前状态
	statusCmd := exec.Command("supervisorctl", "status", "yaf")
	statusOutput, _ := statusCmd.CombinedOutput()
	c.logger.Info("[YAF_RESTART] YAF 当前状态",
		zap.String("status_before", strings.TrimSpace(string(statusOutput))),
	)

	cmd := exec.Command("supervisorctl", "restart", "yaf")
	output, err := cmd.CombinedOutput()
	restartDuration := time.Since(startTime)
	
	if err != nil {
		c.logger.Error("[YAF_RESTART] YAF 重启失败",
			zap.Error(err),
			zap.String("output", string(output)),
			zap.Duration("duration", restartDuration),
		)
		return fmt.Errorf("supervisorctl restart yaf failed: %w, output: %s", err, output)
	}

	outputStr := strings.TrimSpace(string(output))
	c.logger.Info("[YAF_RESTART] YAF 重启成功",
		zap.String("output", outputStr),
		zap.Duration("duration", restartDuration),
	)

	// 等待一小段时间后检查状态
	time.Sleep(1 * time.Second)
	statusCmd = exec.Command("supervisorctl", "status", "yaf")
	statusOutput, _ = statusCmd.CombinedOutput()
	c.logger.Info("[YAF_RESTART] YAF 重启后状态",
		zap.String("status_after", strings.TrimSpace(string(statusOutput))),
	)

	return nil
}

// RestartPipeline 重启 Pipeline 进程
func (c *Controller) RestartPipeline() error {
	startTime := time.Now()
	c.logger.Info("[PIPELINE_RESTART] 开始重启 Pipeline 进程",
		zap.Time("restart_time", startTime),
	)

	// 先检查当前状态
	statusCmd := exec.Command("supervisorctl", "status", "pipeline")
	statusOutput, _ := statusCmd.CombinedOutput()
	c.logger.Info("[PIPELINE_RESTART] Pipeline 当前状态",
		zap.String("status_before", strings.TrimSpace(string(statusOutput))),
	)

	cmd := exec.Command("supervisorctl", "restart", "pipeline")
	output, err := cmd.CombinedOutput()
	restartDuration := time.Since(startTime)
	
	if err != nil {
		c.logger.Error("[PIPELINE_RESTART] Pipeline 重启失败",
			zap.Error(err),
			zap.String("output", string(output)),
			zap.Duration("duration", restartDuration),
		)
		return fmt.Errorf("supervisorctl restart pipeline failed: %w, output: %s", err, output)
	}

	outputStr := strings.TrimSpace(string(output))
	c.logger.Info("[PIPELINE_RESTART] Pipeline 重启成功",
		zap.String("output", outputStr),
		zap.Duration("duration", restartDuration),
	)

	// 等待一小段时间后检查状态
	time.Sleep(1 * time.Second)
	statusCmd = exec.Command("supervisorctl", "status", "pipeline")
	statusOutput, _ = statusCmd.CombinedOutput()
	c.logger.Info("[PIPELINE_RESTART] Pipeline 重启后状态",
		zap.String("status_after", strings.TrimSpace(string(statusOutput))),
	)

	return nil
}

// RestartAll 重启所有相关进程
// 重要：必须先重启 Pipeline（super_mediator），再重启 YAF
// 因为 YAF 需要连接到 super_mediator
func (c *Controller) RestartAll() error {
	totalStartTime := time.Now()
	c.logger.Info("[RESTART_ALL] 开始重启所有进程（Pipeline -> YAF）",
		zap.Time("restart_time", totalStartTime),
	)

	// 1. 先停止 YAF，避免它在 Pipeline 重启期间产生连接错误
	c.logger.Info("[RESTART_ALL] 步骤1: 停止 YAF")
	stopCmd := exec.Command("supervisorctl", "stop", "yaf")
	stopOutput, _ := stopCmd.CombinedOutput()
	c.logger.Info("[RESTART_ALL] YAF 停止结果",
		zap.String("output", strings.TrimSpace(string(stopOutput))),
	)

	// 2. 重启 Pipeline
	c.logger.Info("[RESTART_ALL] 步骤2: 重启 Pipeline")
	if err := c.RestartPipeline(); err != nil {
		c.logger.Error("[RESTART_ALL] Pipeline 重启失败",
			zap.Error(err),
		)
		// 尝试启动 YAF，即使 Pipeline 失败
		exec.Command("supervisorctl", "start", "yaf").Run()
		return err
	}

	// 3. 等待 Pipeline 稳定（super_mediator 启动并监听端口）
	waitDuration := 5 * time.Second
	c.logger.Info("[RESTART_ALL] 步骤3: 等待 Pipeline 稳定",
		zap.Duration("wait_duration", waitDuration),
	)
	time.Sleep(waitDuration)

	// 4. 启动 YAF
	c.logger.Info("[RESTART_ALL] 步骤4: 启动 YAF")
	startCmd := exec.Command("supervisorctl", "start", "yaf")
	startOutput, err := startCmd.CombinedOutput()
	if err != nil {
		c.logger.Error("[RESTART_ALL] YAF 启动失败",
			zap.Error(err),
			zap.String("output", string(startOutput)),
		)
		return fmt.Errorf("supervisorctl start yaf failed: %w, output: %s", err, startOutput)
	}
	c.logger.Info("[RESTART_ALL] YAF 启动结果",
		zap.String("output", strings.TrimSpace(string(startOutput))),
	)

	// 5. 等待 YAF 稳定
	time.Sleep(2 * time.Second)

	totalDuration := time.Since(totalStartTime)
	c.logger.Info("[RESTART_ALL] 所有进程重启完成",
		zap.Duration("total_duration", totalDuration),
	)

	return nil
}

// GetStatus 获取进程状态
func (c *Controller) GetStatus() (string, error) {
	cmd := exec.Command("supervisorctl", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// supervisorctl status 即使进程停止也会返回非 0
		c.logger.Warn("supervisorctl status returned error", zap.Error(err))
	}
	return string(output), nil
}

// CheckSupervisor 检查 supervisord 是否运行
func (c *Controller) CheckSupervisor() bool {
	cmd := exec.Command("supervisorctl", "pid")
	err := cmd.Run()
	return err == nil
}

