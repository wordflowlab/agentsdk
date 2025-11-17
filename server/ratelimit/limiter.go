package ratelimit

import (
	"sync"
	"time"
)

// Limiter 速率限制器接口
type Limiter interface {
	// Allow 检查是否允许请求
	Allow(key string) bool

	// Reset 重置指定 key 的限制
	Reset(key string)

	// GetInfo 获取限制信息
	GetInfo(key string) *LimitInfo
}

// LimitInfo 限制信息
type LimitInfo struct {
	Limit     int       // 限制数量
	Remaining int       // 剩余数量
	ResetAt   time.Time // 重置时间
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // 每秒补充的令牌数
	capacity int           // 桶容量
	window   time.Duration // 清理过期桶的时间窗口
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(rate, capacity int, window time.Duration) *TokenBucketLimiter {
	limiter := &TokenBucketLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		capacity: capacity,
		window:   window,
	}

	// 启动清理 goroutine
	go limiter.cleanup()

	return limiter
}

// Allow 检查是否允许请求
func (l *TokenBucketLimiter) Allow(key string) bool {
	l.mu.Lock()
	b, exists := l.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     float64(l.capacity),
			lastRefill: time.Now(),
		}
		l.buckets[key] = b
	}
	l.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// 补充令牌
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokensToAdd := elapsed * float64(l.rate)
	b.tokens = min(b.tokens+tokensToAdd, float64(l.capacity))
	b.lastRefill = now

	// 尝试消费一个令牌
	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

// Reset 重置指定 key 的限制
func (l *TokenBucketLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

// GetInfo 获取限制信息
func (l *TokenBucketLimiter) GetInfo(key string) *LimitInfo {
	l.mu.RLock()
	b, exists := l.buckets[key]
	l.mu.RUnlock()

	if !exists {
		return &LimitInfo{
			Limit:     l.capacity,
			Remaining: l.capacity,
			ResetAt:   time.Now().Add(time.Second),
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	return &LimitInfo{
		Limit:     l.capacity,
		Remaining: int(b.tokens),
		ResetAt:   b.lastRefill.Add(time.Second),
	}
}

// cleanup 清理过期的桶
func (l *TokenBucketLimiter) cleanup() {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, b := range l.buckets {
			b.mu.Lock()
			if now.Sub(b.lastRefill) > l.window {
				delete(l.buckets, key)
			}
			b.mu.Unlock()
		}
		l.mu.Unlock()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	mu      sync.RWMutex
	windows map[string]*window
	limit   int
	window  time.Duration
}

type window struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(limit int, windowDuration time.Duration) *SlidingWindowLimiter {
	limiter := &SlidingWindowLimiter{
		windows: make(map[string]*window),
		limit:   limit,
		window:  windowDuration,
	}

	// 启动清理 goroutine
	go limiter.cleanup()

	return limiter
}

// Allow 检查是否允许请求
func (l *SlidingWindowLimiter) Allow(key string) bool {
	l.mu.Lock()
	w, exists := l.windows[key]
	if !exists {
		w = &window{
			requests: make([]time.Time, 0),
		}
		l.windows[key] = w
	}
	l.mu.Unlock()

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// 清理过期的请求
	validRequests := make([]time.Time, 0)
	for _, t := range w.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	w.requests = validRequests

	// 检查是否超过限制
	if len(w.requests) < l.limit {
		w.requests = append(w.requests, now)
		return true
	}

	return false
}

// Reset 重置指定 key 的限制
func (l *SlidingWindowLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, key)
}

// GetInfo 获取限制信息
func (l *SlidingWindowLimiter) GetInfo(key string) *LimitInfo {
	l.mu.RLock()
	w, exists := l.windows[key]
	l.mu.RUnlock()

	if !exists {
		return &LimitInfo{
			Limit:     l.limit,
			Remaining: l.limit,
			ResetAt:   time.Now().Add(l.window),
		}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// 计算有效请求数
	validCount := 0
	var oldestRequest time.Time
	for _, t := range w.requests {
		if t.After(cutoff) {
			validCount++
			if oldestRequest.IsZero() || t.Before(oldestRequest) {
				oldestRequest = t
			}
		}
	}

	resetAt := now.Add(l.window)
	if !oldestRequest.IsZero() {
		resetAt = oldestRequest.Add(l.window)
	}

	return &LimitInfo{
		Limit:     l.limit,
		Remaining: l.limit - validCount,
		ResetAt:   resetAt,
	}
}

// cleanup 清理过期的窗口
func (l *SlidingWindowLimiter) cleanup() {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-l.window * 2)

		for key, w := range l.windows {
			w.mu.Lock()
			if len(w.requests) == 0 || (len(w.requests) > 0 && w.requests[len(w.requests)-1].Before(cutoff)) {
				delete(l.windows, key)
			}
			w.mu.Unlock()
		}
		l.mu.Unlock()
	}
}
