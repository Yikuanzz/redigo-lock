package lock

import (
	"context"
	"time"
)

// Lock 分布式锁接口
type Lock interface {
	// Lock 加锁，支持超时设置
	Lock(ctx context.Context, timeout time.Duration) error

	// TryLock 非阻塞尝试加锁
	TryLock(ctx context.Context) (bool, error)

	// Unlock 解锁
	Unlock(ctx context.Context) error

	// IsLocked 检查是否已加锁
	IsLocked(ctx context.Context) bool

	// Refresh 手动刷新锁的过期时间
	Refresh(ctx context.Context) error
}

// LockInfo 锁信息接口
type LockInfo interface {
	// GetOwner 获取锁的持有者信息
	GetOwner() string

	// GetHoldDuration 获取锁的持有时长
	GetHoldDuration() time.Duration

	// GetExpiryTime 获取锁的过期时间
	GetExpiryTime() time.Time
}

// Options 锁配置选项
type Options struct {
	// 基础配置
	ExpireTime       time.Duration
	EnableWatchDog   bool
	WatchDogInterval time.Duration

	// 重试配置
	RetryTimes    int           // 重试次数
	RetryInterval time.Duration // 重试间隔

	// 红锁配置
	EnableRedLock    bool
	RedLockInstances int
	QuorumSize       int // 法定人数，默认为 N/2+1

	// 重入配置
	EnableReentrant bool // 是否启用重入
	MaxReentrantNum int  // 最大重入次数

	// 监控配置
	EnableMetrics bool   // 是否启用指标收集
	MetricsPrefix string // 指标前缀

	// 高级特性
	EnableFencing bool // 是否启用 Fencing Token
	EnableLease   bool // 是否启用租约模式
}

// Option 函数式选项模式
type Option func(*Options)

// WithExpireTime 设置过期时间
func WithExpireTime(expireTime time.Duration) Option {
	return func(o *Options) {
		o.ExpireTime = expireTime
	}
}

// WithWatchDog 设置看门狗模式
func WithWatchDog(enable bool, interval time.Duration) Option {
	return func(o *Options) {
		o.EnableWatchDog = enable
		if interval > 0 {
			o.WatchDogInterval = interval
		}
	}
}

// WithRetry 设置重试策略
func WithRetry(times int, interval time.Duration) Option {
	return func(o *Options) {
		o.RetryTimes = times
		o.RetryInterval = interval
	}
}

// WithRedLock 设置红锁模式
func WithRedLock(enable bool, instances int) Option {
	return func(o *Options) {
		o.EnableRedLock = enable
		o.RedLockInstances = instances
		if instances > 0 {
			o.QuorumSize = instances/2 + 1
		}
	}
}

// WithReentrant 设置重入锁
func WithReentrant(enable bool, maxNum int) Option {
	return func(o *Options) {
		o.EnableReentrant = enable
		o.MaxReentrantNum = maxNum
	}
}

// WithMetrics 设置监控
func WithMetrics(enable bool, prefix string) Option {
	return func(o *Options) {
		o.EnableMetrics = enable
		o.MetricsPrefix = prefix
	}
}

// WithFencing 设置 Fencing Token
func WithFencing(enable bool) Option {
	return func(o *Options) {
		o.EnableFencing = enable
	}
}

// WithLease 设置租约模式
func WithLease(enable bool) Option {
	return func(o *Options) {
		o.EnableLease = enable
	}
}

// DefaultOptions 默认配置
func DefaultOptions() *Options {
	return &Options{
		ExpireTime:       30 * time.Second,
		WatchDogInterval: 10 * time.Second,
		RetryTimes:       3,
		RetryInterval:    time.Second,
		MaxReentrantNum:  1,
		QuorumSize:       2,
	}
}
