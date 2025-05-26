package metrics

import (
	"sync/atomic"
	"time"
)

// Metrics 监控指标
type Metrics struct {
	// 获取锁相关指标
	acquireCount     uint64
	acquireFailCount uint64
	acquireTimeTotal int64
	acquireTimeCount uint64

	// 释放锁相关指标
	releaseCount     uint64
	releaseFailCount uint64

	// 看门狗相关指标
	watchdogRenewCount     uint64
	watchdogRenewFailCount uint64

	// 锁状态指标
	lockTimeoutCount uint64
	lockHoldingTime  int64
}

// NewMetrics 创建监控指标实例
func NewMetrics() *Metrics {
	return &Metrics{}
}

// IncrAcquireCount 增加获取锁次数
func (m *Metrics) IncrAcquireCount() {
	atomic.AddUint64(&m.acquireCount, 1)
}

// IncrAcquireFailCount 增加获取锁失败次数
func (m *Metrics) IncrAcquireFailCount() {
	atomic.AddUint64(&m.acquireFailCount, 1)
}

// AddAcquireTime 添加获取锁耗时
func (m *Metrics) AddAcquireTime(duration time.Duration) {
	atomic.AddInt64(&m.acquireTimeTotal, int64(duration))
	atomic.AddUint64(&m.acquireTimeCount, 1)
}

// IncrReleaseCount 增加释放锁次数
func (m *Metrics) IncrReleaseCount() {
	atomic.AddUint64(&m.releaseCount, 1)
}

// IncrReleaseFailCount 增加释放锁失败次数
func (m *Metrics) IncrReleaseFailCount() {
	atomic.AddUint64(&m.releaseFailCount, 1)
}

// IncrWatchdogRenewCount 增加看门狗续期次数
func (m *Metrics) IncrWatchdogRenewCount() {
	atomic.AddUint64(&m.watchdogRenewCount, 1)
}

// IncrWatchdogRenewFailCount 增加看门狗续期失败次数
func (m *Metrics) IncrWatchdogRenewFailCount() {
	atomic.AddUint64(&m.watchdogRenewFailCount, 1)
}

// IncrLockTimeoutCount 增加锁超时次数
func (m *Metrics) IncrLockTimeoutCount() {
	atomic.AddUint64(&m.lockTimeoutCount, 1)
}

// SetLockHoldingTime 设置锁持有时间
func (m *Metrics) SetLockHoldingTime(duration time.Duration) {
	atomic.StoreInt64(&m.lockHoldingTime, int64(duration))
}

// GetMetrics 获取所有指标
func (m *Metrics) GetMetrics() map[string]interface{} {
	acquireTimeCount := atomic.LoadUint64(&m.acquireTimeCount)
	var avgAcquireTime int64
	if acquireTimeCount > 0 {
		avgAcquireTime = atomic.LoadInt64(&m.acquireTimeTotal) / int64(acquireTimeCount)
	}

	return map[string]interface{}{
		"acquire_count":             atomic.LoadUint64(&m.acquireCount),
		"acquire_fail_count":        atomic.LoadUint64(&m.acquireFailCount),
		"acquire_avg_time_ns":       avgAcquireTime,
		"release_count":             atomic.LoadUint64(&m.releaseCount),
		"release_fail_count":        atomic.LoadUint64(&m.releaseFailCount),
		"watchdog_renew_count":      atomic.LoadUint64(&m.watchdogRenewCount),
		"watchdog_renew_fail_count": atomic.LoadUint64(&m.watchdogRenewFailCount),
		"lock_timeout_count":        atomic.LoadUint64(&m.lockTimeoutCount),
		"lock_holding_time_ns":      atomic.LoadInt64(&m.lockHoldingTime),
	}
}

// Reset 重置所有指标
func (m *Metrics) Reset() {
	atomic.StoreUint64(&m.acquireCount, 0)
	atomic.StoreUint64(&m.acquireFailCount, 0)
	atomic.StoreInt64(&m.acquireTimeTotal, 0)
	atomic.StoreUint64(&m.acquireTimeCount, 0)
	atomic.StoreUint64(&m.releaseCount, 0)
	atomic.StoreUint64(&m.releaseFailCount, 0)
	atomic.StoreUint64(&m.watchdogRenewCount, 0)
	atomic.StoreUint64(&m.watchdogRenewFailCount, 0)
	atomic.StoreUint64(&m.lockTimeoutCount, 0)
	atomic.StoreInt64(&m.lockHoldingTime, 0)
}
