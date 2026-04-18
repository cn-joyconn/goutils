package forward

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// pooledConn 池化连接,封装网络连接并追踪使用状态
type pooledConn struct {
	net.Conn
	createdAt time.Time   // 连接创建时间
	inUse     atomic.Bool // 是否正在使用
}

// markInUse 标记为使用中
func (pc *pooledConn) markInUse() {
	pc.inUse.Store(true)
}

// markIdle 标记为空闲,并更新空闲开始时间
func (pc *pooledConn) markIdle() {
	pc.inUse.Store(false)
	pc.createdAt = time.Now()
}

// ConnPool TCP连接池,复用到目标服务器的连接
type ConnPool struct {
	targetIP    string           // 目标服务器IP
	targetPort  int              // 目标服务器端口
	dialTimeout time.Duration    // 连接超时时间
	maxIdle     int              // 最大空闲连接数
	maxActive   int              // 最大活跃连接数
	idleTimeout time.Duration    // 空闲连接超时时间
	idleConns   chan *pooledConn // 空闲连接 channel
	activeCount atomic.Int32     // 当前活跃连接计数
	mu          sync.Mutex       // 互斥锁,保护并发操作
	cond        *sync.Cond       // 条件变量,用于等待可用连接
	logger      Logger           // 日志记录器
}

// PoolConfig 连接池配置
type PoolConfig struct {
	TargetIP    string        // 目标服务器IP
	TargetPort  int           // 目标服务器端口
	DialTimeout time.Duration // 连接超时
	MaxIdle     int           // 最大空闲连接数
	MaxActive   int           // 最大活跃连接数
	IdleTimeout time.Duration // 空闲超时时间
	Logger      Logger        // 日志记录器
}

// NewConnPool 创建连接池实例
func NewConnPool(config PoolConfig) *ConnPool {
	// 设置默认日志记录器
	if config.Logger == nil {
		config.Logger = NewLogger(LogLevelInfo)
	}
	// 设置默认值
	if config.MaxIdle <= 0 {
		config.MaxIdle = 10
	}
	if config.MaxActive <= 0 {
		config.MaxActive = 100
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 5 * time.Minute
	}
	if config.DialTimeout <= 0 {
		config.DialTimeout = 5 * time.Second
	}

	pool := &ConnPool{
		targetIP:    config.TargetIP,
		targetPort:  config.TargetPort,
		dialTimeout: config.DialTimeout,
		maxIdle:     config.MaxIdle,
		maxActive:   config.MaxActive,
		idleTimeout: config.IdleTimeout,
		idleConns:   make(chan *pooledConn, config.MaxIdle),
		logger:      config.Logger,
	}
	pool.cond = sync.NewCond(&pool.mu)
	return pool
}

// Get 从连接池获取一个连接
// 优先返回空闲连接,无可用时创建新连接,达到最大活跃数时阻塞等待
func (p *ConnPool) Get() (*pooledConn, error) {
	for {
		// 尝试从空闲队列获取连接
		select {
		case pc := <-p.idleConns:
			if pc == nil {
				continue
			}
			// 检查连接是否超时,超时的连接直接关闭
			if time.Since(pc.createdAt) > p.idleTimeout {
				pc.Close()
				continue
			}
			// 检查是否达到最大活跃数
			if p.activeCount.Load() >= int32(p.maxActive) {
				p.idleConns <- pc
				time.Sleep(10 * time.Millisecond)
				continue
			}
			pc.markInUse()
			p.activeCount.Add(1)
			return pc, nil
		default:
		}

		// 达到最大活跃数,等待连接释放
		if p.activeCount.Load() >= int32(p.maxActive) {
			p.mu.Lock()
			p.cond.Wait()
			p.mu.Unlock()
			continue
		}

		// 创建新连接
		conn, err := p.dial()
		if err != nil {
			return nil, err
		}
		pc := &pooledConn{Conn: conn, createdAt: time.Now()}
		pc.markInUse()
		p.activeCount.Add(1)
		return pc, nil
	}
}

// dial 拨号连接到目标服务器
func (p *ConnPool) dial() (net.Conn, error) {
	addr := formatAddr(p.targetIP, p.targetPort)
	return net.DialTimeout("tcp", addr, p.dialTimeout)
}

// Release 释放连接回连接池
// conn为nil时仅减少活跃计数;否则尝试放入空闲队列,队列满时关闭连接
func (p *ConnPool) Release(conn net.Conn) {
	if conn == nil {
		return
	}

	p.activeCount.Add(-1)

	if pc, ok := conn.(*pooledConn); ok {
		pc.markIdle()
		select {
		case p.idleConns <- pc:
			p.cond.Signal()
		default:
			// 空闲队列已满,关闭连接
			conn.Close()
		}
	} else {
		conn.Close()
	}
}

// Close 关闭连接池,清空所有空闲连接
func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.idleConns)
	for pc := range p.idleConns {
		if pc != nil {
			pc.Close()
		}
	}
}

// Stats 返回连接池当前统计信息
func (p *ConnPool) Stats() PoolStats {
	return PoolStats{
		Active:    int(p.activeCount.Load()),
		Idle:      len(p.idleConns),
		MaxIdle:   p.maxIdle,
		MaxActive: p.maxActive,
	}
}

// PoolStats 连接池统计信息
type PoolStats struct {
	Active    int // 当前活跃连接数
	Idle      int // 当前空闲连接数
	MaxIdle   int // 最大空闲连接数
	MaxActive int // 最大活跃连接数
}

func formatAddr(ip string, port int) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Sprintf("%s:%d", ip, port)
	}
	if parsedIP.To4() != nil {
		return fmt.Sprintf("%s:%d", ip, port)
	}
	return fmt.Sprintf("[%s]:%d", ip, port)
}
