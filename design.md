# redigo-lock
在 redigo 基础之上封装实现的分布式锁

## 功能特性 ʕ •ᴥ•ʔ
- 支持通过阻塞轮询和非阻塞模式取锁
- 支持使用看门狗模式进行锁过期时间的自动延期
- 支持使用红锁 redLock 解决 redis 存在的数据弱一致性问题
- 支持锁重入机制
- 支持优雅降级策略
- 支持分布式事务支持
- 支持锁统计和监控

## 详细设计 (◕‿◕✿)

### 基础接口设计
```go
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

// 锁信息接口
type LockInfo interface {
    // GetOwner 获取锁的持有者信息
    GetOwner() string
    
    // GetHoldDuration 获取锁的持有时长
    GetHoldDuration() time.Duration
    
    // GetExpiryTime 获取锁的过期时间
    GetExpiryTime() time.Time
}
```

### 配置选项
```go
type Options struct {
    // 基础配置
    ExpireTime time.Duration
    EnableWatchDog bool
    WatchDogInterval time.Duration
    
    // 重试配置
    RetryTimes int           // 重试次数
    RetryInterval time.Duration  // 重试间隔
    
    // 红锁配置
    EnableRedLock bool
    RedLockInstances int
    QuorumSize int          // 法定人数，默认为 N/2+1
    
    // 重入配置
    EnableReentrant bool    // 是否启用重入
    MaxReentrantNum int    // 最大重入次数
    
    // 监控配置
    EnableMetrics bool     // 是否启用指标收集
    MetricsPrefix string   // 指标前缀
    
    // 降级配置
    FallbackHandler func() error  // 降级处理函数
    
    // 高级特性
    EnableFencing bool    // 是否启用 Fencing Token
    EnableLease bool     // 是否启用租约模式
}
```

### 实现细节

#### 1. 单机模式优化
- 使用 Lua 脚本实现原子性操作
- 引入 Fencing Token 机制防止脑裂
- 支持锁重入计数
- 实现优雅的错误处理和资源释放

#### 2. 看门狗模式增强
- 自适应续期间隔
- 健康检查机制
- 续期失败重试策略
- 支持手动续期操作

#### 3. 红锁模式增强
- 动态节点管理
- 故障转移机制
- 一致性哈希支持
- 支持读写分离

#### 4. 监控指标
```go
type Metrics struct {
    AcquireCount uint64    // 获取锁次数
    AcquireFailCount uint64 // 获取锁失败次数
    ReleaseCount uint64    // 释放锁次数
    WatchdogRenewCount uint64 // 看门狗续期次数
    LockTimeoutCount uint64   // 锁超时次数
    AverageLockTime float64   // 平均持锁时间
}
```

#### 5. 分布式事务支持
```go
type DistributedTransaction interface {
    // Begin 开始事务
    Begin(ctx context.Context) error
    
    // Commit 提交事务
    Commit(ctx context.Context) error
    
    // Rollback 回滚事务
    Rollback(ctx context.Context) error
}
```

## 使用示例 🌟

```go
// 创建锁实例（支持链式调用）
lock := redlock.New("my-lock", redisClient).
    WithExpireTime(30 * time.Second).
    WithWatchDog(true).
    WithRetry(3, time.Second).
    WithMetrics(true).
    Build()

// 使用分布式事务
tx := lock.BeginTransaction(ctx)
defer tx.Rollback(ctx) // 失败时回滚

err := tx.Begin(ctx)
if err != nil {
    return err
}

// 执行业务逻辑
if err := doSomething(); err != nil {
    return err
}

return tx.Commit(ctx)
```

## 性能优化建议 🚀
1. 使用连接池管理 Redis 连接
2. 批量操作使用 pipeline
3. 合理设置超时和重试参数
4. 使用本地缓存减少 Redis 访问

## 注意事项 ⚠️
1. 建议合理设置锁的过期时间，避免因处理时间过长导致锁提前释放
2. 使用看门狗模式时需要注意续期失败的处理
3. 红锁模式下需要保证多个 Redis 实例的时钟同步
4. 建议使用 `defer unlock()` 确保锁的释放
5. 在分布式环境下，需要考虑网络延迟对锁获取的影响
6. 重入锁使用时注意防止死锁
7. 降级策略要考虑业务一致性
8. 监控指标需要定期清理，避免内存泄漏

## 扩展性设计 🔧
1. 插件化的监控系统集成
2. 可扩展的存储引擎支持
3. 自定义序列化方式
4. 事件回调机制
5. 分布式追踪集成