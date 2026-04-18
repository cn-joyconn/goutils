package forward

import "time"

// Config TCP代理配置结构体
type Config struct {
	ID              string        `json:"id"`              // 配置ID标识
	Protocol        string        `json:"protocol"`        // 协议类型: tcp、udp、tcp+udp
	Port            int           `json:"port"`            // 监听端口
	TargetIP        string        `json:"targetIP"`        // 目标服务器IP
	TargetPort      int           `json:"targetPort"`      // 目标服务器端口
	DialTimeout     time.Duration `json:"dialTimeout"`     // 连接目标服务器超时时间
	IdleTimeout     time.Duration `json:"idleTimeout"`     // 连接空闲超时时间
	ReadTimeout     time.Duration `json:"readTimeout"`     // 读取数据超时时间
	WriteTimeout    time.Duration `json:"writeTimeout"`    // 写入数据超时时间
	MaxConn         int           `json:"maxConn"`         // 最大并发连接数
	BufferSize      int           `json:"bufferSize"`      // 数据传输缓冲区大小
	RateLimit       int           `json:"rateLimit"`       // 速率限制(每秒请求数),0表示不限制
	RateLimitBurst  int           `json:"rateLimitBurst"`  // 速率限制突发容量
	MaxRetries      int           `json:"maxRetries"`      // 连接重试次数
	RetryInterval   time.Duration `json:"retryInterval"`   // 重试间隔时间
	PoolMaxIdle     int           `json:"poolMaxIdle"`     // 连接池最大空闲连接数
	PoolMaxActive   int           `json:"poolMaxActive"`   // 连接池最大活跃连接数
	PoolIdleTimeout time.Duration `json:"poolIdleTimeout"` // 连接池连接空闲超时时间
	HealthCheck     bool          `json:"healthCheck"`     // 是否启用健康检查
	HealthInterval  time.Duration `json:"healthInterval"`  // 健康检查间隔时间
	HealthTimeout   time.Duration `json:"healthTimeout"`   // 健康检查超时时间
	LogLevel        LogLevel      `json:"logLevel"`        // 日志级别
	Logger          Logger        `json:"-"`               // 自定义日志记录器,为nil时使用默认logger
	MetricsEnabled  bool          `json:"metricsEnabled"`  // 是否启用Prometheus监控
	MetricsPort     int           `json:"metricsPort"`     // Prometheus监控端口
	UsePool         bool          `json:"usePool"`         // 是否使用连接池
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		DialTimeout:     5 * time.Second,
		IdleTimeout:     30 * time.Second,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		MaxConn:         1000,
		BufferSize:      32 * 1024,
		RateLimit:       0,
		RateLimitBurst:  100,
		MaxRetries:      3,
		RetryInterval:   500 * time.Millisecond,
		PoolMaxIdle:     10,
		PoolMaxActive:   100,
		PoolIdleTimeout: 5 * time.Minute,
		HealthCheck:     true,
		HealthInterval:  10 * time.Second,
		HealthTimeout:   3 * time.Second,
		LogLevel:        LogLevelInfo,
		MetricsEnabled:  false,
		MetricsPort:     9090,
		UsePool:         false,
	}
}

// Validate 验证并填充配置默认值
func (c *Config) Validate() error {
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.TargetIP == "" {
		c.TargetIP = "127.0.0.1"
	}
	if c.TargetPort == 0 {
		c.TargetPort = 80
	}
	return nil
}
