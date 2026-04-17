package test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/cn-joyconn/goutils/gosync"
	"github.com/cn-joyconn/goutils/nets/forward"
)

type ABC interface {
	ID() int64
}
type AA struct {
	B int64
}

func (a *AA) ID() int64 {
	return a.B
}

func TestsyncMap(t *testing.T) {
	mmp := gosync.NewSyncMap[int, ABC]()
	mmp.Put(0, &AA{})
	v, o := mmp.Get(0)
	if o {
		fmt.Println(v.ID())
	}
	mmp.Remove(0)
	v, o = mmp.Get(0)
	if o {
		fmt.Println(v)
	}
}

func Test_TcpForward(t *testing.T) {
	ListenAddr := "8080"
	TargetAddr := "192.168.1.100"
	TargetPort := 80
	config := forward.Config{
		Protocol:     "tcp",
		Port:         ListenAddr,
		TargetIP:     TargetAddr,
		TargetPort:   TargetPort,
		DialTimeout:  5 * time.Second,  // 连接超时5秒
		IdleTimeout:  30 * time.Second, // 空闲超时30秒
		ReadTimeout:  10 * time.Second, // 读取超时10秒
		WriteTimeout: 10 * time.Second, // 写入超时10秒
		MaxConn:      1000,             // 最大连接数
	}

	proxy := forward.NewTCPProxy(config)

	// 启动连接监控
	go proxy.MonitorConnections()

	// 启动代理
	if err := proxy.Start(); err != nil {
		log.Fatalf("代理启动失败: %v", err)
	}
}
