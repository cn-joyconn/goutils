package forward

import (
	"net"
	"sync/atomic"
	"time"
)

// HealthChecker 健康检查器,定期检测目标服务器是否可达
type HealthChecker struct {
	targetIP   string        // 目标服务器IP
	targetPort int           // 目标服务器端口
	interval   time.Duration // 健康检查间隔
	timeout    time.Duration // 健康检查超时
	healthy    atomic.Bool   // 目标服务器健康状态
	logger     Logger        // 日志记录器
}

// HealthConfig 健康检查器配置
type HealthConfig struct {
	TargetIP   string        // 目标服务器IP
	TargetPort int           // 目标服务器端口
	Interval   time.Duration // 检查间隔
	Timeout    time.Duration // 检查超时
	Logger     Logger        // 日志记录器
}

// NewHealthChecker 创建健康检查器实例
func NewHealthChecker(config HealthConfig) *HealthChecker {
	// 设置默认日志记录器
	if config.Logger == nil {
		config.Logger = NewLogger(LogLevelInfo)
	}
	// 设置默认检查间隔
	if config.Interval <= 0 {
		config.Interval = 10 * time.Second
	}
	// 设置默认检查超时
	if config.Timeout <= 0 {
		config.Timeout = 3 * time.Second
	}

	hc := &HealthChecker{
		targetIP:   config.TargetIP,
		targetPort: config.TargetPort,
		interval:   config.Interval,
		timeout:    config.Timeout,
		logger:     config.Logger,
	}
	// 初始状态设为健康
	hc.healthy.Store(true)
	return hc
}

// Start 启动健康检查循环
// ctx 接收停止信号,当关闭时检查器停止运行
func (hc *HealthChecker) Start(ctx <-chan struct{}) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// 启动时立即执行一次检查
	hc.check()

	for {
		select {
		case <-ctx:
			return
		case <-ticker.C:
			hc.check()
		}
	}
}

// check 执行一次健康检查,尝试连接目标服务器
func (hc *HealthChecker) check() {
	addr := formatAddr(hc.targetIP, hc.targetPort)
	conn, err := net.DialTimeout("tcp", addr, hc.timeout)
	if err != nil {
		hc.healthy.Store(false)
		hc.logger.Warn("Health check failed: %v", err)
		return
	}
	conn.Close()
	hc.healthy.Store(true)
}

// IsHealthy 返回目标服务器当前健康状态
func (hc *HealthChecker) IsHealthy() bool {
	return hc.healthy.Load()
}
