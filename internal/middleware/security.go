// Package middleware 安全头 + 速率限制中间件
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders 安全头中间件 (Helmet 等效)
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// Content-Security-Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self' ws: wss:")
		// Strict-Transport-Security — only in non-dev mode
		if gin.Mode() != gin.DebugMode {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Next()
	}
}

// --- Simplified In-Memory Rate Limiter ---

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string]*rateEntry
}

type rateEntry struct {
	count    int
	windowStart time.Time
}

var globalRateLimiter = &rateLimiter{
	requests: make(map[string]*rateEntry),
}

// RateLimit 基于 IP 的速率限制中间件
// maxRequests: 窗口内最大请求数, window: 时间窗口
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	cleanupInterval := window * 2
	return func(c *gin.Context) {
		ip := c.ClientIP()

		globalRateLimiter.mu.Lock()
		entry, exists := globalRateLimiter.requests[ip]
		now := time.Now()

		// Periodic cleanup to prevent memory leak
		if now.UnixMilli()%int64(cleanupInterval.Milliseconds()) < 100 {
			go globalRateLimiter.cleanup(cleanupInterval)
		}

		if !exists || now.Sub(entry.windowStart) > window {
			globalRateLimiter.requests[ip] = &rateEntry{
				count:       1,
				windowStart: now,
			}
			globalRateLimiter.mu.Unlock()
			c.Next()
			return
		}

		if entry.count >= maxRequests {
			globalRateLimiter.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		entry.count++
		globalRateLimiter.mu.Unlock()
		c.Next()
	}
}

// cleanup 清理过期条目防止内存泄漏
func (rl *rateLimiter) cleanup(olderThan time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	threshold := time.Now().Add(-olderThan)
	for ip, entry := range rl.requests {
		if entry.windowStart.Before(threshold) {
			delete(rl.requests, ip)
		}
	}
}

// NewRateLimiter 创建一个独立的速率限制器实例
func NewRateLimiter() *rateLimiter {
	return &rateLimiter{
		requests: make(map[string]*rateEntry),
	}
}

// AuthRateLimit 登录接口专用速率限制 (5次/60秒)
var authLimiter = NewRateLimiter()

func AuthRateLimit() gin.HandlerFunc {
	cleanupInterval := 120 * time.Second
	return func(c *gin.Context) {
		ip := c.ClientIP()
		authLimiter.mu.Lock()
		entry, exists := authLimiter.requests[ip]
		now := time.Now()
		if now.UnixMilli()%int64(cleanupInterval.Milliseconds()) < 100 {
			go authLimiter.cleanup(cleanupInterval)
		}
		if !exists || now.Sub(entry.windowStart) > 60*time.Second {
			authLimiter.requests[ip] = &rateEntry{count: 1, windowStart: now}
			authLimiter.mu.Unlock()
			c.Next()
			return
		}
		if entry.count >= 5 {
			authLimiter.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "登录尝试过于频繁，请60秒后再试",
			})
			c.Abort()
			return
		}
		entry.count++
		authLimiter.mu.Unlock()
		c.Next()
	}
}
