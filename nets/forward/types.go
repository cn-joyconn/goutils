package forward

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// LogLevel 日志级别枚举
type LogLevel int

const (
	LogLevelDebug LogLevel = iota // 调试级别
	LogLevelInfo                  // 信息级别
	LogLevelWarn                  // 警告级别
	LogLevelError                 // 错误级别
)

// String 返回日志级别字符串表示
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志接口,定义日志输出方法
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// defaultLogger 基于标准log包的默认日志实现
type defaultLogger struct {
	level LogLevel // 日志级别,低于此级别的日志不输出
}

// NewLogger 创建日志记录器实例
func NewLogger(level LogLevel) Logger {
	return &defaultLogger{level: level}
}

// Debug 输出调试级别日志
func (l *defaultLogger) Debug(format string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info 输出信息级别日志
func (l *defaultLogger) Info(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		log.Printf("[INFO] "+format, args...)
	}
}

// Warn 输出警告级别日志
func (l *defaultLogger) Warn(format string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		log.Printf("[WARN] "+format, args...)
	}
}

// Error 输出错误级别日志
func (l *defaultLogger) Error(format string, args ...interface{}) {
	if l.level <= LogLevelError {
		log.Printf("[ERROR] "+format, args...)
	}
}

// connection TCP代理会话连接,封装客户端和目标服务器连接对
type connection struct {
	id         string       // 连接唯一标识符
	clientConn net.Conn     // 客户端连接
	targetConn net.Conn     // 目标服务器连接
	poolRef    ConnReleaser // 连接池引用,用于释放连接
	createdAt  time.Time    // 连接创建时间
	lastActive time.Time    // 最后活跃时间,用于空闲检测
	mu         sync.Mutex   // 保护lastActive的互斥锁
	closed     atomic.Bool  // 连接是否已关闭标记
}

// ConnReleaser 连接释放器接口,用于连接池释放连接
type ConnReleaser interface {
	Release(conn net.Conn)
}

// updateActivity 更新最后活跃时间
func (c *connection) updateActivity() {
	c.mu.Lock()
	c.lastActive = time.Now()
	c.mu.Unlock()
}

// isIdle 判断连接是否处于空闲状态
// idleTimeout 空闲超时时间,超过此时间未活动则视为空闲
func (c *connection) isIdle(idleTimeout time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.lastActive) > idleTimeout
}

// Close 关闭代理连接,释放相关资源
// 使用CAS确保只关闭一次
func (c *connection) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	var errs []error
	// 关闭客户端连接
	if c.clientConn != nil {
		if err := c.clientConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	// 关闭目标服务器连接
	if c.targetConn != nil {
		if err := c.targetConn.Close(); err != nil {
			errs = append(errs, err)
		}
	} else if c.poolRef != nil {
		// 如果有连接池引用,释放连接到池中
		c.poolRef.Release(nil)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Read 从客户端读取数据,并更新活跃时间
func (c *connection) Read(p []byte) (n int, err error) {
	n, err = c.clientConn.Read(p)
	if n > 0 {
		c.updateActivity()
	}
	return
}

// Write 写入数据到目标服务器
func (c *connection) Write(p []byte) (n int, err error) {
	return c.targetConn.Write(p)
}

// LocalAddr 返回本地地址
func (c *connection) LocalAddr() net.Addr {
	return c.clientConn.LocalAddr()
}

// RemoteAddr 返回客户端远程地址
func (c *connection) RemoteAddr() net.Addr {
	return c.clientConn.RemoteAddr()
}

// SetDeadline 设置读写操作的截止时间
func (c *connection) SetDeadline(t time.Time) error {
	return c.clientConn.SetDeadline(t)
}

// SetReadDeadline 设置读操作的截止时间
func (c *connection) SetReadDeadline(t time.Time) error {
	return c.clientConn.SetReadDeadline(t)
}

// SetWriteDeadline 设置写操作的截止时间
func (c *connection) SetWriteDeadline(t time.Time) error {
	return c.clientConn.SetWriteDeadline(t)
}

// RateLimiter 令牌桶速率限制器,实现流量控制
type RateLimiter struct {
	rate     int          // 每秒产生的令牌数
	burst    int          // 令牌桶容量
	tokens   atomic.Int64 // 当前令牌数
	lastTick atomic.Int64 // 上次更新时间(纳秒)
}

// NewRateLimiter 创建速率限制器
// rate 每秒令牌数,burst 令牌桶容量
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:  rate,
		burst: burst,
	}
	// 初始令牌数设为桶容量
	rl.tokens.Store(int64(burst))
	rl.lastTick.Store(time.Now().UnixNano())
	return rl
}

// Allow 检查是否允许一个请求通过
func (rl *RateLimiter) Allow() bool {
	return rl.AllowN(1)
}

// AllowN 检查是否允许N个请求通过
// 使用CAS保证原子性,成功返回true,失败返回false
func (rl *RateLimiter) AllowN(n int) bool {
	for {
		now := time.Now().UnixNano()
		lastTick := rl.lastTick.Load()

		// 计算距离上次更新的时间增量
		elapsed := now - lastTick
		// 根据时间增量计算应补充的令牌数
		tokensToAdd := int64(elapsed) * int64(rl.rate) / int64(time.Second)

		currentTokens := rl.tokens.Load()
		newTokens := currentTokens + tokensToAdd
		// 令牌数不能超过桶容量
		if newTokens > int64(rl.burst) {
			newTokens = int64(rl.burst)
		}

		// 令牌不足,拒绝请求
		if newTokens < int64(n) {
			return false
		}

		// CAS更新令牌数和更新时间
		if rl.lastTick.CompareAndSwap(lastTick, now) &&
			rl.tokens.CompareAndSwap(currentTokens, newTokens-int64(n)) {
			return true
		}
	}
}

// Wait 等待获取令牌,最多等待100毫秒
func (rl *RateLimiter) Wait() bool {
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			return true
		}
		time.Sleep(time.Millisecond * 10)
	}
	return false
}

// isTimeoutError 判断错误是否为超时错误
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
