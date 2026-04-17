package forward

import "time"

// Config 代理配置
type Config struct {
	ID           string        `json:"id"`
	Protocol     string        `json:"protocol"`     //协议类型 tcp、udp、tcp\udp
	Port         string        `json:"port"`         // 监听地址
	TargetIP     string        `json:"targetIP"`     // 目标地址
	TargetPort   int           `json:"targetPort"`   // 目标端口
	DialTimeout  time.Duration `json:"dialTimeout"`  // 连接超时
	IdleTimeout  time.Duration `json:"idleTimeout"`  // 空闲超时
	ReadTimeout  time.Duration `json:"readTimeout"`  // 读取超时
	WriteTimeout time.Duration `json:"writeTimeout"` // 写入超时
	MaxConn      int           `json:"maxConn"`      // 最大连接数
}
