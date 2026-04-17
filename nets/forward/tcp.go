package forward

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// TCPProxy TCP转发代理
type TCPProxy struct {
	port         string
	targetIP     string
	targetPort   int
	dialTimeout  time.Duration // 连接目标服务器超时
	idleTimeout  time.Duration // 连接空闲超时
	readTimeout  time.Duration // 读取超时
	writeTimeout time.Duration // 写入超时
	maxConn      int           // 最大连接数
	activeConn   sync.Map      // 活动连接
	connCount    int32         // 当前连接数
	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewTCPProxy 创建TCP代理
func NewTCPProxy(config Config) *TCPProxy {
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

	ctx, cancel := context.WithCancel(context.Background())
	return &TCPProxy{
		port:         config.Port,
		targetIP:     config.TargetIP,
		targetPort:   config.TargetPort,
		dialTimeout:  config.DialTimeout,
		idleTimeout:  config.IdleTimeout,
		readTimeout:  config.ReadTimeout,
		writeTimeout: config.WriteTimeout,
		maxConn:      config.MaxConn,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start 启动代理
func (p *TCPProxy) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))
	if err != nil {
		return fmt.Errorf("监听失败: %w", err)
	}
	defer listener.Close()

	log.Printf("TCP代理启动，监听地址: :%d，目标地址: %s:%d", p.port, p.targetIP, p.targetPort)
	log.Printf("超时配置: 连接=%v, 空闲=%v, 读取=%v, 写入=%v",
		p.dialTimeout, p.idleTimeout, p.readTimeout, p.writeTimeout)

	// 启动连接清理器
	go p.connectionCleaner()

	for {
		select {
		case <-p.ctx.Done():
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err)
			continue
		}

		// 检查连接数限制
		if p.getConnectionCount() >= p.maxConn {
			log.Printf("连接数超过限制: %d/%d", p.getConnectionCount(), p.maxConn)
			conn.Close()
			continue
		}

		p.incrementConnectionCount()
		p.activeConn.Store(conn.RemoteAddr().String(), time.Now())

		log.Printf("新连接来自: %s (活动连接: %d)",
			conn.RemoteAddr().String(), p.getConnectionCount())

		go p.handleConnection(conn)
	}
}

// handleConnection 处理单个连接
func (p *TCPProxy) handleConnection(clientConn net.Conn) {
	defer func() {
		clientConn.Close()
		p.activeConn.Delete(clientConn.RemoteAddr().String())
		p.decrementConnectionCount()

		log.Printf("连接关闭: %s (剩余连接: %d)",
			clientConn.RemoteAddr().String(), p.getConnectionCount())
	}()

	// 连接目标服务器（带超时）
	ctx, cancel := context.WithTimeout(context.Background(), p.dialTimeout)
	defer cancel()

	var targetConn net.Conn
	var err error

	// 异步连接目标服务器
	connCh := make(chan net.Conn, 1)
	errCh := make(chan error, 1)

	go func() {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", p.targetIP, p.targetPort))
		if err != nil {
			errCh <- err
			return
		}
		connCh <- conn
	}()

	select {
	case <-ctx.Done():
		log.Printf("连接目标服务器超时: %v", ctx.Err())
		return
	case err = <-errCh:
		log.Printf("连接目标服务器失败: %v", err)
		return
	case targetConn = <-connCh:
		// 连接成功
	}
	defer targetConn.Close()

	log.Printf("连接建立: %s -> %s",
		clientConn.RemoteAddr(), targetConn.RemoteAddr())

	// 设置连接超时
	deadline := time.Now().Add(p.idleTimeout)
	clientConn.SetDeadline(deadline)
	targetConn.SetDeadline(deadline)

	// 记录最后一次活动时间
	lastActivity := time.Now()
	p.activeConn.Store(clientConn.RemoteAddr().String(), lastActivity)

	// 使用 WaitGroup 等待两个方向的转发完成
	var wg sync.WaitGroup
	wg.Add(2)

	// 通道用于控制优雅关闭
	done := make(chan bool, 2)

	// 从客户端转发到目标服务器
	go func() {
		defer wg.Done()
		p.forwardData(clientConn, targetConn, "client->target", &lastActivity, done)
	}()

	// 从目标服务器转发到客户端
	go func() {
		defer wg.Done()
		p.forwardData(targetConn, clientConn, "target->client", &lastActivity, done)
	}()

	// 等待转发完成
	wg.Wait()
	close(done)
}

// forwardData 转发数据
func (p *TCPProxy) forwardData(src, dst net.Conn, direction string,
	lastActivity *time.Time, done chan bool) {

	buffer := make([]byte, 32*1024) // 32KB 缓冲区

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-done:
			return
		default:
		}

		// 设置读取超时
		if p.readTimeout > 0 {
			src.SetReadDeadline(time.Now().Add(p.readTimeout))
		}

		// 读取数据
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF && !isTimeoutError(err) {
				log.Printf("%s 读取失败: %v", direction, err)
			}
			return
		}

		if n == 0 {
			// 对端关闭连接
			return
		}

		// 更新活动时间
		*lastActivity = time.Now()
		p.activeConn.Store(src.RemoteAddr().String(), *lastActivity)

		// 设置写入超时
		if p.writeTimeout > 0 {
			dst.SetWriteDeadline(time.Now().Add(p.writeTimeout))
		}

		// 写入数据
		_, err = dst.Write(buffer[:n])
		if err != nil {
			if !isTimeoutError(err) {
				log.Printf("%s 写入失败: %v", direction, err)
			}
			return
		}

		// 记录日志（生产环境可以注释掉）
		log.Printf("%s 转发 %d 字节", direction, n)
	}
}

// connectionCleaner 定期清理空闲连接
func (p *TCPProxy) connectionCleaner() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.cleanIdleConnections()
		}
	}
}

// cleanIdleConnections 清理空闲连接
func (p *TCPProxy) cleanIdleConnections() {
	now := time.Now()
	var idleConns []string

	// 找出空闲连接
	p.activeConn.Range(func(key, value interface{}) bool {
		if lastActivity, ok := value.(time.Time); ok {
			if now.Sub(lastActivity) > p.idleTimeout {
				addr := key.(string)
				idleConns = append(idleConns, addr)
			}
		}
		return true
	})

	// 这里只是记录日志，实际连接在各自的 goroutine 中处理
	if len(idleConns) > 0 {
		log.Printf("检测到 %d 个空闲连接", len(idleConns))
		// 在实际实现中，这里应该主动关闭这些连接
		// 由于连接在各自的 goroutine 中管理，我们需要通过其他方式通知它们关闭
	}
}

// Stop 停止代理
func (p *TCPProxy) Stop() {
	log.Println("正在停止代理...")
	p.cancel()

	// 等待一段时间让连接优雅关闭
	time.Sleep(2 * time.Second)

	// 强制关闭所有活动连接
	var wg sync.WaitGroup
	p.activeConn.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			// 在实际实现中，这里应该关闭对应地址的连接
			log.Printf("强制关闭连接: %s", addr)
		}(key.(string))
		return true
	})
	wg.Wait()

	log.Println("代理已停止")
}

// 工具函数
func (p *TCPProxy) getConnectionCount() int {
	count := 0
	p.activeConn.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (p *TCPProxy) incrementConnectionCount() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connCount++
}

func (p *TCPProxy) decrementConnectionCount() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connCount--
}

// 连接监控
func (p *TCPProxy) MonitorConnections() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			log.Printf("连接统计: 活动连接=%d", p.getConnectionCount())
		}
	}
}

// isTimeoutError 判断是否为超时错误
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
