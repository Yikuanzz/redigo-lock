package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yikuanzz/redigo-lock/pkg/lock"
)

func redlockExample() {
	// 创建多个 Redis 实例客户端
	clients := []*redis.Client{
		redis.NewClient(&redis.Options{
			Addr: "localhost:6379", // 实例 1
		}),
		redis.NewClient(&redis.Options{
			Addr: "localhost:6380", // 实例 2
		}),
		redis.NewClient(&redis.Options{
			Addr: "localhost:6381", // 实例 3
		}),
		redis.NewClient(&redis.Options{
			Addr: "localhost:6382", // 实例 4
		}),
		redis.NewClient(&redis.Options{
			Addr: "localhost:6383", // 实例 5
		}),
	}

	// 创建 RedLock
	redLock := lock.NewRedisLock(
		clients[0], // 使用第一个实例作为主实例
		"my-redlock-key",
		lock.WithExpireTime(30*time.Second),
		lock.WithRedLock(true, 5), // 启用 RedLock 模式，使用 5 个实例
		lock.WithWatchDog(true, 10*time.Second),
		lock.WithRetry(3, time.Second),
		lock.WithMetrics(true, "my-redlock"),
	)

	// 获取锁
	ctx := context.Background()
	if err := redLock.Lock(ctx, 10*time.Second); err != nil {
		log.Fatalf("Failed to acquire RedLock: %v", err)
	}

	// 确保锁会被释放
	defer redLock.Unlock(ctx)

	// 检查锁状态
	if redLock.IsLocked(ctx) {
		log.Println("Successfully acquired RedLock! 🔒")
	}

	// 模拟业务处理
	log.Println("Doing some work with RedLock protection...")
	time.Sleep(5 * time.Second)

	// 手动刷新锁（虽然有看门狗，但展示手动刷新的用法）
	if err := redLock.Refresh(ctx); err != nil {
		log.Printf("Warning: Failed to refresh RedLock: %v", err)
	}

	// 完成业务处理
	log.Println("Work done! RedLock will be released... 🔓")
}

func main() {
	log.Println("Starting RedLock example... ʕ •ᴥ•ʔ")
	redlockExample()
	log.Println("RedLock example completed! (｡◕‿◕｡)")
}
