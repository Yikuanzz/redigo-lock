package cluster

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/yikuanzz/redigo-lock/pkg/lock"
)

func createRedisClusterClient() *redis.ClusterClient {
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"localhost:7001",
			"localhost:7002",
			"localhost:7003",
		},
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		DialTimeout:  5 * time.Second,
		PoolSize:     10,
		MinIdleConns: 3,
		MaxRetries:   3,
	})
}

func TestRedisClusterLock(t *testing.T) {
	// 创建 Redis 集群客户端
	client := createRedisClusterClient()
	defer client.Close()

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to Redis cluster: %v", err)
	}

	t.Run("基本加锁解锁", func(t *testing.T) {
		const lockKey = "test-cluster-lock-basic"

		// 清理可能存在的锁
		if err := client.Del(ctx, lockKey).Err(); err != nil {
			t.Logf("Warning: Failed to clean lock: %v", err)
		}

		// 创建锁实例
		redisLock := lock.NewRedisLock(
			client,
			lockKey,
			lock.WithExpireTime(30*time.Second),
		)

		// 获取锁
		err := redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err, "Failed to acquire lock")

		// 验证锁是否存在
		exists, err := client.Exists(ctx, lockKey).Result()
		assert.NoError(t, err, "Failed to check lock existence")
		assert.Equal(t, int64(1), exists, "Lock should exist")

		// 释放锁
		err = redisLock.Unlock(ctx)
		assert.NoError(t, err, "Failed to release lock")

		// 验证锁是否已释放
		exists, err = client.Exists(ctx, lockKey).Result()
		assert.NoError(t, err, "Failed to check lock existence")
		assert.Equal(t, int64(0), exists, "Lock should not exist")
	})

	t.Run("并发测试", func(t *testing.T) {
		const (
			goroutineCount = 5
			lockKey        = "test-cluster-lock-concurrent"
		)
		var counter int
		var mu sync.Mutex
		var wg sync.WaitGroup

		// 清理可能存在的锁
		if err := client.Del(ctx, lockKey).Err(); err != nil {
			t.Logf("Warning: Failed to clean lock: %v", err)
		}

		for i := 0; i < goroutineCount; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				redisLock := lock.NewRedisLock(
					client,
					lockKey,
					lock.WithExpireTime(30*time.Second),
					lock.WithRetry(3, 500*time.Millisecond),
				)

				err := redisLock.Lock(ctx, 10*time.Second)
				if err != nil {
					t.Logf("Goroutine %d failed to acquire lock: %v", id, err)
					return
				}

				defer func() {
					if err := redisLock.Unlock(ctx); err != nil {
						t.Logf("Goroutine %d failed to release lock: %v", id, err)
					}
				}()

				// 验证锁是否存在
				exists, err := client.Exists(ctx, lockKey).Result()
				if err != nil {
					t.Errorf("Goroutine %d: Failed to check lock existence: %v", id, err)
					return
				}
				if exists != 1 {
					t.Errorf("Goroutine %d: Lock should exist", id)
					return
				}

				// 模拟业务操作
				mu.Lock()
				currentValue := counter
				time.Sleep(50 * time.Millisecond)
				counter = currentValue + 1
				mu.Unlock()
			}(i)
		}

		wg.Wait()
		assert.Equal(t, goroutineCount, counter, "Counter should be equal to the number of goroutines")
	})

	t.Run("故障转移测试", func(t *testing.T) {
		const lockKey = "test-cluster-lock-failover"

		// 清理可能存在的锁
		if err := client.Del(ctx, lockKey).Err(); err != nil {
			t.Logf("Warning: Failed to clean lock: %v", err)
		}

		// 创建锁实例
		redisLock := lock.NewRedisLock(
			client,
			lockKey,
			lock.WithExpireTime(5*time.Second),
			lock.WithRetry(3, 500*time.Millisecond),
		)

		// 获取锁
		err := redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err, "Failed to acquire initial lock")

		// 验证锁是否存在
		exists, err := client.Exists(ctx, lockKey).Result()
		assert.NoError(t, err, "Failed to check lock existence")
		assert.Equal(t, int64(1), exists, "Lock should exist")

		// 等待锁过期
		time.Sleep(6 * time.Second)

		// 验证锁是否已过期
		exists, err = client.Exists(ctx, lockKey).Result()
		assert.NoError(t, err, "Failed to check lock existence")
		assert.Equal(t, int64(0), exists, "Lock should have expired")

		// 重新获取锁
		err = redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err, "Failed to acquire lock after expiration")

		// 释放锁
		err = redisLock.Unlock(ctx)
		assert.NoError(t, err, "Failed to release lock")
	})

	t.Run("看门狗自动续期", func(t *testing.T) {
		const lockKey = "test-cluster-lock-watchdog"

		// 清理可能存在的锁
		if err := client.Del(ctx, lockKey).Err(); err != nil {
			t.Logf("Warning: Failed to clean lock: %v", err)
		}

		// 创建锁实例
		redisLock := lock.NewRedisLock(
			client,
			lockKey,
			lock.WithExpireTime(2*time.Second),
			lock.WithWatchDog(true, 1*time.Second),
		)

		// 获取锁
		err := redisLock.Lock(ctx, 5*time.Second)
		assert.NoError(t, err, "Failed to acquire lock")

		// 等待超过原始过期时间
		time.Sleep(3 * time.Second)

		// 验证锁是否仍然存在（因为看门狗会自动续期）
		exists, err := client.Exists(ctx, lockKey).Result()
		assert.NoError(t, err, "Failed to check lock existence")
		assert.Equal(t, int64(1), exists, "Lock should still exist due to watchdog")

		// 释放锁
		err = redisLock.Unlock(ctx)
		assert.NoError(t, err, "Failed to release lock")
	})
}
