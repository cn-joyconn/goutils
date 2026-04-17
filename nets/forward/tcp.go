package forward

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TCPProxy TCP代理,支持连接池、健康检查、速率限制和监控
type TCPProxy struct {
	config        Config                 // 配置信息
	port          string                 // 监听端口
	targetIP      string                 // 目标服务器IP
	targetPort    int                    // 目标服务器端口
	dialTimeout   time.Duration          // 连接超时
	idleTimeout   time.Duration          // 空闲超时
	readTimeout   time.Duration          // 读取超时
	writeTimeout  time.Duration          // 写入超时
	maxConn       int                    // 最大连接数
	bufferSize    int                    // 缓冲区大小
	activeConns   map[string]*connection // 当前活跃连接集合
	connMu        sync.Mutex             // 保护activeConns的互斥锁
	rateLimiter   *RateLimiter           // 速率限制器
	pool          *ConnPool              // 连接池
	healthChecker *HealthChecker         // 健康检查器
	logger        Logger                 // 日志记录器
	metrics       *Metrics               // 监控指标收集器
	ctx           context.Context        // 上下文,用于优雅关闭
	cancel        context.CancelFunc     // 取消函数
	healthy       atomic.Bool            // 整体健康状态
}

// NewTCPProxy 创建TCP代理实例
// config 代理配置,包含监听端口、目标地址、超时设置等参数
func NewTCPProxy(config Config) *TCPProxy {
	// 验证配置
	if err := config.Validate(); err != nil {
		panic(err)
	}

	// 设置默认值
	if config.BufferSize <= 0 {
		config.BufferSize = 32 * 1024
	}
	if config.DialTimeout <= 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 30 * time.Second
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 10 * time.Second
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 10 * time.Second
	}
	if config.MaxConn <= 0 {
		config.MaxConn = 1000
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = 500 * time.Millisecond
	}

	// 创建日志记录器
	var logger Logger
	if config.Logger != nil {
		logger = config.Logger
	} else {
		logger = NewLogger(config.LogLevel)
	}

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 初始化代理结构体
	proxy := &TCPProxy{
		config:       config,
		port:         config.Port,
		targetIP:     config.TargetIP,
		targetPort:   config.TargetPort,
		dialTimeout:  config.DialTimeout,
		idleTimeout:  config.IdleTimeout,
		readTimeout:  config.ReadTimeout,
		writeTimeout: config.WriteTimeout,
		maxConn:      config.MaxConn,
		bufferSize:   config.BufferSize,
		activeConns:  make(map[string]*connection),
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 初始化速率限制器
	if config.RateLimit > 0 {
		if config.RateLimitBurst <= 0 {
			config.RateLimitBurst = 100
		}
		proxy.rateLimiter = NewRateLimiter(config.RateLimit, config.RateLimitBurst)
	}

	// 初始化监控指标
	if config.MetricsEnabled {
		proxy.metrics = NewMetrics("tcpproxy")
	}

	// 初始化连接池
	if config.UsePool {
		proxy.pool = NewConnPool(PoolConfig{
			TargetIP:    config.TargetIP,
			TargetPort:  config.TargetPort,
			DialTimeout: config.DialTimeout,
			MaxIdle:     config.PoolMaxIdle,
			MaxActive:   config.PoolMaxActive,
			IdleTimeout: config.PoolIdleTimeout,
			Logger:      logger,
		})
	}

	// 初始化健康检查器
	if config.HealthCheck {
		if config.HealthInterval <= 0 {
			config.HealthInterval = 10 * time.Second
		}
		if config.HealthTimeout <= 0 {
			config.HealthTimeout = 3 * time.Second
		}
		proxy.healthChecker = NewHealthChecker(HealthConfig{
			TargetIP:   config.TargetIP,
			TargetPort: config.TargetPort,
			Interval:   config.HealthInterval,
			Timeout:    config.HealthTimeout,
			Logger:     logger,
		})
		// 启动健康检查循环
		healthCtx := make(chan struct{})
		go proxy.healthChecker.Start(healthCtx)
		// 代理停止时关闭健康检查
		go func() {
			<-ctx.Done()
			close(healthCtx)
		}()
	}

	proxy.healthy.Store(true)

	return proxy
}

// Start 启动TCP代理服务,阻塞直到收到停止信号
func (p *TCPProxy) Start() error {
	addr := fmt.Sprintf(":%s", p.port)
	if p.port == "" {
		addr = ":8080"
	}

	// 创建TCP监听器
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	// 输出启动信息
	p.logger.Info("TCPProxy started, listening on %s, forwarding to %s:%d",
		addr, p.targetIP, p.targetPort)
	p.logger.Info("Config: dial=%v, idle=%v, read=%v, write=%v, maxConn=%d, buffer=%d",
		p.dialTimeout, p.idleTimeout, p.readTimeout, p.writeTimeout, p.maxConn, p.bufferSize)

	// 输出速率限制配置
	if p.rateLimiter != nil {
		p.logger.Info("Rate limiting enabled: %d req/s, burst=%d",
			p.config.RateLimit, p.config.RateLimitBurst)
	}
	// 输出连接池配置
	if p.pool != nil {
		p.logger.Info("Connection pool enabled: maxIdle=%d, maxActive=%d",
			p.config.PoolMaxIdle, p.config.PoolMaxActive)
	}

	// 启动空闲连接清理器
	go p.connectionCleaner()

	// 接受连接主循环
	for {
		select {
		case <-p.ctx.Done():
			return nil
		default:
		}

		// 接受客户端连接
		clientConn, err := listener.Accept()
		if err != nil {
			p.logger.Error("Accept error: %v", err)
			continue
		}

		// 速率限制检查
		if p.rateLimiter != nil && !p.rateLimiter.Allow() {
			if p.metrics != nil {
				p.metrics.IncRateLimitRejects()
			}
			p.logger.Warn("Connection rejected due to rate limiting: %s", clientConn.RemoteAddr())
			clientConn.Close()
			continue
		}

		// 检查最大连接数限制
		p.connMu.Lock()
		connCount := len(p.activeConns)
		p.connMu.Unlock()

		if connCount >= p.maxConn {
			p.logger.Warn("Connection rejected: maxConn reached (%d/%d)", connCount, p.maxConn)
			clientConn.Close()
			continue
		}

		// 健康检查
		if p.healthChecker != nil && !p.healthChecker.IsHealthy() {
			p.logger.Warn("Connection rejected: target unhealthy")
			clientConn.Close()
			continue
		}

		// 处理连接
		go p.handleConnection(clientConn)
	}
}

// handleConnection 处理单个客户端连接
// 创建到目标服务器的连接并双向转发数据
func (p *TCPProxy) handleConnection(clientConn net.Conn) {
	startTime := time.Now()
	addr := clientConn.RemoteAddr().String()

	// 更新监控指标
	if p.metrics != nil {
		p.metrics.IncConnections()
		defer func() {
			p.metrics.DecConnections()
			p.metrics.RecordConnectionDuration(time.Since(startTime).Seconds())
		}()
	}

	// 创建代理连接对象
	conn := &connection{
		id:         addr,
		clientConn: clientConn,
		createdAt:  startTime,
		lastActive: startTime,
	}

	// 注册到活跃连接集合
	p.connMu.Lock()
	p.activeConns[addr] = conn
	p.connMu.Unlock()

	// 连接关闭时清理
	defer func() {
		p.connMu.Lock()
		delete(p.activeConns, addr)
		p.connMu.Unlock()

		duration := time.Since(startTime)
		conn.Close()
		p.logger.Info("Connection closed: %s (duration: %v, remaining: %d)",
			addr, duration, len(p.activeConns))
	}()

	// 建立到目标服务器的连接
	var targetConn net.Conn
	var err error

	if p.pool != nil {
		// 使用连接池获取连接
		targetConn, err = p.getConnWithRetry()
		conn.poolRef = p.pool
	} else {
		// 直接拨号
		targetConn, err = p.dialWithRetry()
	}

	if err != nil {
		p.logger.Error("Failed to connect to target: %v", err)
		if p.metrics != nil {
			p.metrics.RecordError("dial")
		}
		return
	}

	conn.targetConn = targetConn

	// 记录拨号耗时
	if p.metrics != nil {
		p.metrics.RecordDial(time.Since(startTime).Seconds())
	}

	p.logger.Info("Connection established: %s -> %s", addr, targetConn.RemoteAddr())

	// 设置空闲超时
	deadline := time.Now().Add(p.idleTimeout)
	clientConn.SetDeadline(deadline)
	targetConn.SetDeadline(deadline)

	// 双向数据转发
	var wg sync.WaitGroup
	wg.Add(2)

	stopChan := make(chan struct{})

	// 客户端 -> 目标服务器
	go func() {
		defer wg.Done()
		p.forwardData(clientConn, targetConn, "client->target", conn)
		close(stopChan)
	}()

	// 目标服务器 -> 客户端
	go func() {
		defer wg.Done()
		p.forwardData(targetConn, clientConn, "target->client", conn)
		select {
		case <-stopChan:
		default:
			close(stopChan)
		}
	}()

	wg.Wait()
}

// dialWithRetry 拨号连接到目标服务器,失败时重试
func (p *TCPProxy) dialWithRetry() (net.Conn, error) {
	var lastErr error
	addr := fmt.Sprintf("%s:%d", p.targetIP, p.targetPort)
	dialer := &net.Dialer{Timeout: p.dialTimeout}

	for i := 0; i < p.config.MaxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), p.dialTimeout)
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		cancel()

		if err == nil {
			return conn, nil
		}
		lastErr = err

		if i < p.config.MaxRetries-1 {
			p.logger.Warn("Dial attempt %d failed: %v, retrying...", i+1, err)
			time.Sleep(p.config.RetryInterval)
		}
	}

	return nil, fmt.Errorf("dial failed after %d attempts: %w", p.config.MaxRetries, lastErr)
}

// getConnWithRetry 从连接池获取连接,失败时重试
func (p *TCPProxy) getConnWithRetry() (*pooledConn, error) {
	var lastErr error

	for i := 0; i < p.config.MaxRetries; i++ {
		conn, err := p.pool.Get()
		if err == nil {
			return conn, nil
		}
		lastErr = err

		if i < p.config.MaxRetries-1 {
			p.logger.Warn("Pool get attempt %d failed: %v, retrying...", i+1, err)
			time.Sleep(p.config.RetryInterval)
		}
	}

	return nil, fmt.Errorf("pool get failed after %d attempts: %w", p.config.MaxRetries, lastErr)
}

// forwardData 单向数据转发
// src 数据源, dst 数据目的地, direction 方向标识
// 返回时若检测到异常(非超时),会通知关闭连接
func (p *TCPProxy) forwardData(src, dst net.Conn, direction string, conn *connection) {
	buffer := make([]byte, p.bufferSize)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		// 设置读取超时
		if p.readTimeout > 0 {
			src.SetReadDeadline(time.Now().Add(p.readTimeout))
		}

		// 从源读取数据
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF && !isTimeoutError(err) {
				p.logger.Debug("%s read error: %v", direction, err)
				if p.metrics != nil {
					p.metrics.RecordError("read")
				}
			}
			// 任何读取错误都关闭两个连接以快速终止
			conn.Close()
			return
		}

		if n == 0 {
			// 读到0字节(正常EOF或空读),也关闭连接
			conn.Close()
			return
		}

		// 更新连接活跃时间
		conn.updateActivity()

		// 设置写入超时
		if p.writeTimeout > 0 {
			dst.SetWriteDeadline(time.Now().Add(p.writeTimeout))
		}

		// 写入目标
		written, err := dst.Write(buffer[:n])
		if err != nil {
			if !isTimeoutError(err) {
				p.logger.Debug("%s write error: %v", direction, err)
				if p.metrics != nil {
					p.metrics.RecordError("write")
				}
			}
			// 写入错误也关闭连接
			conn.Close()
			return
		}

		// 记录传输字节数
		if p.metrics != nil {
			p.metrics.RecordBytes(direction, int64(written))
		}
	}
}

// connectionCleaner 空闲连接清理循环,定期检测并关闭空闲连接
func (p *TCPProxy) connectionCleaner() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.cleanIdleConnections()
			// 更新连接池统计
			if p.pool != nil && p.metrics != nil {
				p.metrics.UpdatePoolStats(p.pool.Stats())
			}
		}
	}
}

// cleanIdleConnections 检测并关闭空闲连接
func (p *TCPProxy) cleanIdleConnections() {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	now := time.Now()
	var toClose []*connection

	// 找出所有空闲连接
	for _, conn := range p.activeConns {
		if conn.isIdle(p.idleTimeout) {
			toClose = append(toClose, conn)
		}
	}

	// 关闭空闲连接
	for _, conn := range toClose {
		p.logger.Info("Closing idle connection: %s (idle for %v)",
			conn.id, now.Sub(conn.lastActive))
		conn.Close()
		delete(p.activeConns, conn.id)
	}
}

// Stop 停止代理服务,关闭所有连接
func (p *TCPProxy) Stop() {
	p.logger.Info("Stopping TCPProxy...")
	p.cancel()

	// 等待一段时间让连接优雅关闭
	time.Sleep(2 * time.Second)

	// 强制关闭所有活跃连接
	p.connMu.Lock()
	for _, conn := range p.activeConns {
		conn.Close()
	}
	p.activeConns = make(map[string]*connection)
	p.connMu.Unlock()

	// 关闭连接池
	if p.pool != nil {
		p.pool.Close()
	}

	p.logger.Info("TCPProxy stopped")
}

// GetActiveConnections 返回当前活跃连接数
func (p *TCPProxy) GetActiveConnections() int {
	p.connMu.Lock()
	defer p.connMu.Unlock()
	return len(p.activeConns)
}

// IsHealthy 返回代理健康状态
func (p *TCPProxy) IsHealthy() bool {
	if p.healthChecker != nil {
		return p.healthChecker.IsHealthy()
	}
	return p.healthy.Load()
}
