package lock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/yikuanzz/redigo-lock/pkg/errors"
	"github.com/yikuanzz/redigo-lock/pkg/metrics"
)

const (
	// 默认锁前缀
	defaultLockPrefix = "redigo_lock:"
	// 默认锁值前缀
	defaultValuePrefix = "owner:"
)

// RedisClient 接口定义，支持单机和集群模式
type RedisClient interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
}

// RedisLock Redis 分布式锁实现
type RedisLock struct {
	// 锁配置
	key     string
	value   string
	options *Options

	// Redis 客户端（支持单机和集群）
	client RedisClient

	// 看门狗相关
	watchdogCtx    context.Context
	watchdogCancel context.CancelFunc
	watchdogWg     sync.WaitGroup

	// 重入相关
	reentrantCount int
	mutex          sync.Mutex

	// 监控相关
	metrics *metrics.Metrics

	// 日志
	logger *zap.Logger
}

// NewRedisLock 创建 Redis 分布式锁
func NewRedisLock(client RedisClient, key string, opts ...Option) *RedisLock {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	return &RedisLock{
		key:     defaultLockPrefix + key,
		value:   defaultValuePrefix + uuid.New().String(),
		options: options,
		client:  client,
		metrics: metrics.NewMetrics(),
		logger:  logger,
	}
}

// Lock 加锁
func (l *RedisLock) Lock(ctx context.Context, timeout time.Duration) error {
	startTime := time.Now()
	defer func() {
		l.metrics.AddAcquireTime(time.Since(startTime))
	}()

	// 检查是否可以重入
	if l.options.EnableReentrant {
		l.mutex.Lock()
		if l.reentrantCount > 0 && l.IsLocked(ctx) {
			if l.reentrantCount >= l.options.MaxReentrantNum {
				l.mutex.Unlock()
				l.metrics.IncrAcquireFailCount()
				return errors.ErrMaxReentrantExceeded
			}
			l.reentrantCount++
			l.mutex.Unlock()
			l.metrics.IncrAcquireCount()
			return nil
		}
		l.mutex.Unlock()
	}

	// 尝试获取锁
	success, err := l.tryAcquireLock(ctx, timeout)
	if err != nil {
		l.metrics.IncrAcquireFailCount()
		return err
	}

	if !success {
		l.metrics.IncrAcquireFailCount()
		return errors.ErrLockAcquireFailed
	}

	// 获取锁成功，增加重入计数
	if l.options.EnableReentrant {
		l.mutex.Lock()
		l.reentrantCount = 1
		l.mutex.Unlock()
	}

	// 启动看门狗
	if l.options.EnableWatchDog {
		l.startWatchdog()
	}

	l.metrics.IncrAcquireCount()
	return nil
}

// TryLock 非阻塞尝试加锁
func (l *RedisLock) TryLock(ctx context.Context) (bool, error) {
	startTime := time.Now()
	defer func() {
		l.metrics.AddAcquireTime(time.Since(startTime))
	}()

	// 检查是否可以重入
	if l.options.EnableReentrant {
		l.mutex.Lock()
		if l.reentrantCount > 0 {
			if l.reentrantCount >= l.options.MaxReentrantNum {
				l.mutex.Unlock()
				l.metrics.IncrAcquireFailCount()
				return false, errors.ErrMaxReentrantExceeded
			}
			l.reentrantCount++
			l.mutex.Unlock()
			l.metrics.IncrAcquireCount()
			return true, nil
		}
		l.mutex.Unlock()
	}

	// 尝试获取锁
	success, err := l.tryAcquireLock(ctx, 0)
	if err != nil {
		l.metrics.IncrAcquireFailCount()
		return false, err
	}

	if success {
		// 启动看门狗
		if l.options.EnableWatchDog {
			l.startWatchdog()
		}
		l.metrics.IncrAcquireCount()
	} else {
		l.metrics.IncrAcquireFailCount()
	}

	return success, nil
}

// Unlock 解锁
func (l *RedisLock) Unlock(ctx context.Context) error {
	// 检查重入计数
	if l.options.EnableReentrant {
		l.mutex.Lock()
		if l.reentrantCount > 1 {
			l.reentrantCount--
			l.mutex.Unlock()
			l.metrics.IncrReleaseCount()
			return nil
		}
		if l.reentrantCount == 1 {
			l.reentrantCount = 0
		}
		l.mutex.Unlock()
	}

	// 如果是重入锁且计数已经为 0，说明已经被完全释放
	if l.options.EnableReentrant && l.reentrantCount == 0 {
		l.metrics.IncrReleaseCount()
		return nil
	}

	// 停止看门狗
	if l.options.EnableWatchDog {
		l.stopWatchdog()
	}

	// 使用 Lua 脚本确保原子性释放
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Int64()
	if err != nil {
		l.metrics.IncrReleaseFailCount()
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result == 0 {
		l.metrics.IncrReleaseFailCount()
		return errors.ErrLockNotHeld
	}

	l.metrics.IncrReleaseCount()
	return nil
}

// IsLocked 检查是否已加锁
func (l *RedisLock) IsLocked(ctx context.Context) bool {
	value, err := l.client.Get(ctx, l.key).Result()
	if err != nil {
		return false
	}
	return value == l.value
}

// Refresh 手动刷新锁的过期时间
func (l *RedisLock) Refresh(ctx context.Context) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value, l.options.ExpireTime.Milliseconds()).Int64()
	if err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	if result == 0 {
		return errors.ErrLockNotHeld
	}

	return nil
}

// tryAcquireLock 尝试获取锁
func (l *RedisLock) tryAcquireLock(ctx context.Context, timeout time.Duration) (bool, error) {
	if timeout == 0 {
		return l.client.SetNX(ctx, l.key, l.value, l.options.ExpireTime).Result()
	}
	return l.tryAcquireLockWithRetry(ctx, timeout)
}

// tryAcquireLockWithRetry 带重试的获取锁
func (l *RedisLock) tryAcquireLockWithRetry(ctx context.Context, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		success, err := l.client.SetNX(ctx, l.key, l.value, l.options.ExpireTime).Result()
		if err != nil {
			return false, err
		}
		if success {
			return true, nil
		}

		if time.Now().After(deadline) {
			return false, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(l.options.RetryInterval):
			continue
		}
	}
}

// startWatchdog 启动看门狗
func (l *RedisLock) startWatchdog() {
	l.watchdogCtx, l.watchdogCancel = context.WithCancel(context.Background())
	l.watchdogWg.Add(1)

	go func() {
		defer l.watchdogWg.Done()
		ticker := time.NewTicker(l.options.WatchDogInterval)
		defer ticker.Stop()

		for {
			select {
			case <-l.watchdogCtx.Done():
				return
			case <-ticker.C:
				if err := l.Refresh(l.watchdogCtx); err != nil {
					l.logger.Error("watchdog refresh failed", zap.Error(err))
					return
				}
			}
		}
	}()
}

// stopWatchdog 停止看门狗
func (l *RedisLock) stopWatchdog() {
	if l.watchdogCancel != nil {
		l.watchdogCancel()
		l.watchdogWg.Wait()
	}
}
