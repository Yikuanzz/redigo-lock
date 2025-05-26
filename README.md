# Redigo Lock

基于 Redigo 实现的分布式锁 SDK，提供简单易用的 API 和丰富的功能特性。

## 功能特性

- 支持阻塞和非阻塞模式获取锁
- 支持看门狗自动续期
- 支持 RedLock 算法
- 支持锁重入
- 支持优雅降级
- 支持监控指标收集
- 支持分布式事务

## 安装

```bash
go get github.com/yikuanzz/redigo-lock
```

## 快速开始

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/gomodule/redigo/redis"
    "github.com/linuxz/redigo-lock/pkg/lock"
)

func main() {
    // 创建 Redis 连接池
    pool := &redis.Pool{
        MaxIdle:     3,
        IdleTimeout: 240 * time.Second,
        Dial: func() (redis.Conn, error) {
            return redis.Dial("tcp", "localhost:6379")
        },
    }

    // 创建分布式锁
    redisLock := lock.NewRedisLock(
        pool,
        "my-lock",
        lock.WithExpireTime(30*time.Second),
        lock.WithWatchDog(true, 10*time.Second),
    )

    // 获取锁
    ctx := context.Background()
    if err := redisLock.Lock(ctx, 5*time.Second); err != nil {
        log.Fatal(err)
    }
    defer redisLock.Unlock(ctx)

    // 执行业务逻辑
    log.Println("获取锁成功，执行业务逻辑...")
}
```

## 高级特性

### 1. 看门狗模式

自动续期，防止锁过期：

```go
lock := NewRedisLock(
    pool,
    "my-lock",
    WithExpireTime(30*time.Second),
    WithWatchDog(true, 10*time.Second),
)
```

### 2. 重入锁

支持同一个持有者多次获取锁：

```go
lock := NewRedisLock(
    pool,
    "my-lock",
    WithReentrant(true, 3),
)
```

### 3. RedLock 模式

使用多个 Redis 实例保证强一致性：

```go
lock := NewRedisLock(
    pool,
    "my-lock",
    WithRedLock(true, 5),
)
```

### 4. 监控指标

收集锁的使用情况：

```go
lock := NewRedisLock(
    pool,
    "my-lock",
    WithMetrics(true, "my-lock"),
)

// 获取指标
metrics := lock.GetMetrics()
```

## 注意事项

1. 合理设置锁的过期时间，避免业务执行时间超过锁的过期时间
2. 使用 defer 确保锁的释放
3. 在分布式环境下注意时钟同步
4. 合理使用重试机制，避免无效等待
5. 监控指标需要定期清理，避免内存泄漏

## 性能优化

1. 使用连接池管理 Redis 连接
2. 批量操作使用 pipeline
3. 合理设置超时和重试参数
4. 使用本地缓存减少 Redis 访问

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License 