package single

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/yikuanzz/redigo-lock/pkg/lock"
)

func TestSingleRedisLock(t *testing.T) {
	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	t.Run("基本加锁解锁", func(t *testing.T) {
		redisLock := lock.NewRedisLock(
			client,
			"test-lock",
			lock.WithExpireTime(30*time.Second),
		)

		ctx := context.Background()
		err := redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err)

		isLocked := redisLock.IsLocked(ctx)
		assert.True(t, isLocked)

		err = redisLock.Unlock(ctx)
		assert.NoError(t, err)

		isLocked = redisLock.IsLocked(ctx)
		assert.False(t, isLocked)
	})

	t.Run("锁超时", func(t *testing.T) {
		redisLock := lock.NewRedisLock(
			client,
			"test-lock-timeout",
			lock.WithExpireTime(1*time.Second),
		)

		ctx := context.Background()
		err := redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err)

		// 等待锁过期
		time.Sleep(2 * time.Second)

		isLocked := redisLock.IsLocked(ctx)
		assert.False(t, isLocked)
	})

	t.Run("看门狗自动续期", func(t *testing.T) {
		redisLock := lock.NewRedisLock(
			client,
			"test-lock-watchdog",
			lock.WithExpireTime(1*time.Second),
			lock.WithWatchDog(true, 500*time.Millisecond),
		)

		ctx := context.Background()
		err := redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err)

		// 等待超过原始过期时间
		time.Sleep(2 * time.Second)

		// 锁应该还在，因为看门狗会自动续期
		isLocked := redisLock.IsLocked(ctx)
		assert.True(t, isLocked)

		err = redisLock.Unlock(ctx)
		assert.NoError(t, err)
	})

	t.Run("重入锁", func(t *testing.T) {
		redisLock := lock.NewRedisLock(
			client,
			"test-lock-reentrant",
			lock.WithExpireTime(30*time.Second),
			lock.WithReentrant(true, 2),
		)

		ctx := context.Background()

		// 第一次加锁
		err := redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err)

		// 第二次加锁（重入）
		err = redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err)

		// 第三次加锁应该失败（超过最大重入次数）
		err = redisLock.Lock(ctx, 5*time.Second)
		assert.Error(t, err)

		// 解锁两次
		err = redisLock.Unlock(ctx)
		assert.NoError(t, err)
		err = redisLock.Unlock(ctx)
		assert.NoError(t, err)
	})

	t.Run("并发测试", func(t *testing.T) {
		const (
			goroutineCount = 10
			lockKey        = "test-lock-concurrent"
		)
		var counter int
		var mu sync.Mutex
		var wg sync.WaitGroup

		// 确保测试开始前锁是释放的
		cleanLock := lock.NewRedisLock(client, lockKey)
		cleanLock.Unlock(context.Background())

		for i := 0; i < goroutineCount; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				redisLock := lock.NewRedisLock(
					client,
					lockKey,
					lock.WithExpireTime(30*time.Second),
					lock.WithRetry(3, 100*time.Millisecond),
				)

				ctx := context.Background()
				err := redisLock.Lock(ctx, 5*time.Second)
				if err != nil {
					t.Logf("Goroutine %d failed to acquire lock: %v", id, err)
					return
				}

				// 确保锁会被释放
				defer func() {
					if err := redisLock.Unlock(ctx); err != nil {
						t.Logf("Goroutine %d failed to release lock: %v", id, err)
					}
				}()

				// 验证当前 goroutine 确实持有锁
				if !redisLock.IsLocked(ctx) {
					t.Errorf("Goroutine %d doesn't hold the lock after Lock()", id)
					return
				}

				// 模拟业务操作
				mu.Lock()
				currentValue := counter
				time.Sleep(10 * time.Millisecond) // 模拟一些工作
				counter = currentValue + 1
				mu.Unlock()
			}(i)
		}

		wg.Wait()
		assert.Equal(t, goroutineCount, counter, "Counter should be equal to the number of goroutines")
	})
}
