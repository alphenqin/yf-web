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
	c.logger.Info("restarting yaf via supervisorctl")

	cmd := exec.Command("supervisorctl", "restart", "yaf")
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.Error("failed to restart yaf",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return fmt.Errorf("supervisorctl restart yaf failed: %w, output: %s", err, output)
	}

	c.logger.Info("yaf restarted successfully", zap.String("output", strings.TrimSpace(string(output))))
	return nil
}

// RestartPipeline 重启 Pipeline 进程
func (c *Controller) RestartPipeline() error {
	c.logger.Info("restarting pipeline via supervisorctl")

	cmd := exec.Command("supervisorctl", "restart", "pipeline")
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.Error("failed to restart pipeline",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return fmt.Errorf("supervisorctl restart pipeline failed: %w, output: %s", err, output)
	}

	c.logger.Info("pipeline restarted successfully", zap.String("output", strings.TrimSpace(string(output))))
	return nil
}

// RestartAll 重启所有相关进程
func (c *Controller) RestartAll() error {
	// 先重启 YAF
	if err := c.RestartYAF(); err != nil {
		return err
	}

	// 等待一小段时间
	time.Sleep(2 * time.Second)

	// 再重启 Pipeline
	if err := c.RestartPipeline(); err != nil {
		return err
	}

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

