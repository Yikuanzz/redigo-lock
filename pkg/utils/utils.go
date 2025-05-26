package utils

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	b := make([]byte, length/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetCurrentTimeMillis 获取当前时间戳（毫秒）
func GetCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// GetCurrentTimeMicros 获取当前时间戳（微秒）
func GetCurrentTimeMicros() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// Sleep 休眠指定时间
func Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// RetryWithTimeout 带超时的重试函数
func RetryWithTimeout(fn func() error, retryTimes int, retryInterval, timeout time.Duration) error {
	done := make(chan error, 1)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	go func() {
		var err error
		for i := 0; i < retryTimes; i++ {
			if err = fn(); err == nil {
				done <- nil
				return
			}
			time.Sleep(retryInterval)
		}
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-timer.C:
		return ErrTimeout
	}
}

// ErrTimeout 超时错误
var ErrTimeout = timeoutError("timeout")

type timeoutError string

func (e timeoutError) Error() string { return string(e) }

// IsTimeout 判断是否为超时错误
func IsTimeout(err error) bool {
	_, ok := err.(timeoutError)
	return ok
}
