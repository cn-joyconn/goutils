package forward

import (
	"fmt"
	"runtime"
	"sync/atomic"
)

type Metrics struct {
	activeConnections  atomic.Int64
	bytesFromClient    atomic.Int64
	bytesToClient      atomic.Int64
	totalConnections   atomic.Int64
	errors             [3]atomic.Int64
	dialDurationSum    atomic.Int64
	dialCount          atomic.Int64
	forwardDurationSum [2]atomic.Int64
	forwardCount       [2]atomic.Int64
	connDurationSum    atomic.Int64
	connCount          atomic.Int64
	rateLimitRejects   atomic.Int64
	poolActive         atomic.Int64
	poolIdle           atomic.Int64
}

const (
	errTypeDial = iota
	errTypeRead
	errTypeWrite
)

const (
	dirClientToTarget = iota
	dirTargetToClient
)

func NewMetrics(namespace string) *Metrics {
	return &Metrics{}
}

func (m *Metrics) IncConnections() {
	m.activeConnections.Add(1)
	m.totalConnections.Add(1)
}

func (m *Metrics) DecConnections() {
	m.activeConnections.Add(-1)
}

func (m *Metrics) RecordBytes(direction string, n int64) {
	switch direction {
	case "client->target":
		m.bytesFromClient.Add(n)
	case "target->client":
		m.bytesToClient.Add(n)
	}
}

func (m *Metrics) RecordError(errType string) {
	var idx int
	switch errType {
	case "dial":
		idx = errTypeDial
	case "read":
		idx = errTypeRead
	case "write":
		idx = errTypeWrite
	}
	m.errors[idx].Add(1)
}

func (m *Metrics) RecordDial(duration float64) {
	m.dialDurationSum.Add(int64(duration * 1e9))
	m.dialCount.Add(1)
}

func (m *Metrics) RecordForward(direction string, duration float64) {
	var idx int
	if direction == "client->target" {
		idx = dirClientToTarget
	} else {
		idx = dirTargetToClient
	}
	m.forwardDurationSum[idx].Add(int64(duration * 1e9))
	m.forwardCount[idx].Add(1)
}

func (m *Metrics) RecordConnectionDuration(duration float64) {
	m.connDurationSum.Add(int64(duration * 1e9))
	m.connCount.Add(1)
}

func (m *Metrics) IncRateLimitRejects() {
	m.rateLimitRejects.Add(1)
}

func (m *Metrics) UpdatePoolStats(stats PoolStats) {
	m.poolActive.Store(int64(stats.Active))
	m.poolIdle.Store(int64(stats.Idle))
}

type MetricsSnapshot struct {
	ActiveConnections  int64
	BytesFromClient    int64
	BytesToClient      int64
	TotalConnections   int64
	ErrorsDial         int64
	ErrorsRead         int64
	ErrorsWrite        int64
	DialAvgMs          float64
	ForwardClientAvgMs float64
	ForwardTargetAvgMs float64
	ConnAvgMs          float64
	RateLimitRejects   int64
	PoolActive         int64
	PoolIdle           int64
	NumGoroutine       int
	MemAllocMB         uint64
	MemSysMB           uint64
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	var dialAvg, fwdClientAvg, fwdTargetAvg, connAvg float64

	dialCount := m.dialCount.Load()
	if dialCount > 0 {
		dialAvg = float64(m.dialDurationSum.Load()) / float64(dialCount) / 1e6
	}

	fwdClientCount := m.forwardCount[dirClientToTarget].Load()
	if fwdClientCount > 0 {
		fwdClientAvg = float64(m.forwardDurationSum[dirClientToTarget].Load()) / float64(fwdClientCount) / 1e6
	}

	fwdTargetCount := m.forwardCount[dirTargetToClient].Load()
	if fwdTargetCount > 0 {
		fwdTargetAvg = float64(m.forwardDurationSum[dirTargetToClient].Load()) / float64(fwdTargetCount) / 1e6
	}

	connCount := m.connCount.Load()
	if connCount > 0 {
		connAvg = float64(m.connDurationSum.Load()) / float64(connCount) / 1e6
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return MetricsSnapshot{
		ActiveConnections:  m.activeConnections.Load(),
		BytesFromClient:    m.bytesFromClient.Load(),
		BytesToClient:      m.bytesToClient.Load(),
		TotalConnections:   m.totalConnections.Load(),
		ErrorsDial:         m.errors[errTypeDial].Load(),
		ErrorsRead:         m.errors[errTypeRead].Load(),
		ErrorsWrite:        m.errors[errTypeWrite].Load(),
		DialAvgMs:          dialAvg,
		ForwardClientAvgMs: fwdClientAvg,
		ForwardTargetAvgMs: fwdTargetAvg,
		ConnAvgMs:          connAvg,
		RateLimitRejects:   m.rateLimitRejects.Load(),
		PoolActive:         m.poolActive.Load(),
		PoolIdle:           m.poolIdle.Load(),
		NumGoroutine:       runtime.NumGoroutine(),
		MemAllocMB:         ms.Alloc / 1024 / 1024,
		MemSysMB:           ms.Sys / 1024 / 1024,
	}
}

func (s MetricsSnapshot) String() string {
	return "Metrics{" +
		"activeConns=" + formatInt(s.ActiveConnections) +
		", totalConns=" + formatInt(s.TotalConnections) +
		", bytesIn=" + formatInt(s.BytesFromClient) +
		", bytesOut=" + formatInt(s.BytesToClient) +
		", errors={dial=" + formatInt(s.ErrorsDial) +
		", read=" + formatInt(s.ErrorsRead) +
		", write=" + formatInt(s.ErrorsWrite) + "}" +
		", dialAvgMs=" + formatFloat(s.DialAvgMs) +
		", fwdClientAvgMs=" + formatFloat(s.ForwardClientAvgMs) +
		", fwdTargetAvgMs=" + formatFloat(s.ForwardTargetAvgMs) +
		", connAvgMs=" + formatFloat(s.ConnAvgMs) +
		", rateLimitRejects=" + formatInt(s.RateLimitRejects) +
		", pool={active=" + formatInt(s.PoolActive) +
		", idle=" + formatInt(s.PoolIdle) + "}" +
		", goroutines=" + formatInt(int64(s.NumGoroutine)) +
		", memAllocMB=" + formatUint(s.MemAllocMB) +
		", memSysMB=" + formatUint(s.MemSysMB) +
		"}"
}

func formatInt(v int64) string   { return fmt.Sprintf("%d", v) }
func formatUint(v uint64) string { return fmt.Sprintf("%d", v) }
func formatFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
