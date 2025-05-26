package errors

import "github.com/pkg/errors"

var (
	// ErrLockNotFound 锁不存在
	ErrLockNotFound = errors.New("lock not found")
	// ErrLockAcquireFailed 获取锁失败
	ErrLockAcquireFailed = errors.New("failed to acquire lock")
	// ErrLockReleaseFailed 释放锁失败
	ErrLockReleaseFailed = errors.New("failed to release lock")
	// ErrLockTimeout 获取锁超时
	ErrLockTimeout = errors.New("lock acquire timeout")
	// ErrLockNotHeld 未持有锁
	ErrLockNotHeld = errors.New("lock not held")
	// ErrMaxReentrantExceeded 超过最大重入次数
	ErrMaxReentrantExceeded = errors.New("max reentrant count exceeded")
	// ErrInvalidExpireTime 无效的过期时间
	ErrInvalidExpireTime = errors.New("invalid expire time")
	// ErrWatchdogRenewFailed 看门狗续期失败
	ErrWatchdogRenewFailed = errors.New("watchdog renew failed")
	// ErrRedLockQuorumNotMet 红锁法定人数未达到
	ErrRedLockQuorumNotMet = errors.New("redlock quorum not met")
)
