package test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/cn-joyconn/goutils/nets/forward"
)

func TestTcpForward(t *testing.T) {
	ListenAddr := "8080"
	TargetAddr := "127.0.0.1"
	TargetPort := 60001

	config := forward.DefaultConfig()
	config.Port = ListenAddr
	config.TargetIP = TargetAddr
	config.TargetPort = TargetPort
	config.MaxRetries = 3
	config.HealthCheck = false
	config.UsePool = false
	config.RateLimit = 1000
	config.RateLimitBurst = 100
	proxy := forward.NewTCPProxy(config)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Printf("Active connections: %d, Healthy: %v\n",
				proxy.GetActiveConnections(), proxy.IsHealthy())
		}
	}()

	if err := proxy.Start(); err != nil {
		log.Fatalf("Proxy start failed: %v", err)
	}
}

func TestTcpForwardWithPool(t *testing.T) {
	config := forward.DefaultConfig()
	config.Port = "8081"
	config.TargetIP = "127.0.0.1"
	config.TargetPort = 80
	config.UsePool = true
	config.PoolMaxIdle = 5
	config.PoolMaxActive = 50
	config.HealthCheck = true
	config.MetricsEnabled = true

	proxy := forward.NewTCPProxy(config)

	if err := proxy.Start(); err != nil {
		log.Fatalf("Proxy start failed: %v", err)
	}
}

func TestTcpForwardByMyLog(t *testing.T) {
	ListenAddr := "8080"
	TargetAddr := "127.0.0.1"
	TargetPort := 60001

	config := forward.DefaultConfig()
	config.Port = ListenAddr
	config.TargetIP = TargetAddr
	config.TargetPort = TargetPort
	config.MaxRetries = 3
	config.HealthCheck = false
	config.UsePool = true
	config.DialTimeout = 5 * time.Second
	config.IdleTimeout = 120 * time.Second
	config.ReadTimeout = 60 * time.Second
	config.WriteTimeout = 60 * time.Second
	config.RateLimit = 1000
	config.RateLimitBurst = 100
	config.Logger = &myLogger{}
	proxy := forward.NewTCPProxy(config)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Printf("Active connections: %d, Healthy: %v\n",
				proxy.GetActiveConnections(), proxy.IsHealthy())
		}
	}()

	if err := proxy.Start(); err != nil {
		log.Fatalf("Proxy start failed: %v", err)
	}
}

// 实现 Logger 接口
type myLogger struct{}

func (l *myLogger) Debug(format string, args ...interface{}) {
	fmt.Printf("tcp formward DebugLog:%s", format)
}
func (l *myLogger) Info(format string, args ...interface{}) {
	fmt.Printf("tcp formward InfoLog:%s", format)
}
func (l *myLogger) Warn(format string, args ...interface{}) {
	fmt.Printf("tcp formward WarnLog:%s", format)
}
func (l *myLogger) Error(format string, args ...interface{}) {
	fmt.Printf("tcp formward ErrorLog:%s", format)
}
