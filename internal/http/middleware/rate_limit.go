package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"avagostar-form-server/internal/utils"
	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	limit int
	mu    sync.Mutex
	items map[string]*rateEntry
}

type rateEntry struct {
	count int
	reset time.Time
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		limit: limit,
		items: make(map[string]*rateEntry),
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()
		entry, ok := rl.items[ip]
		if !ok || now.After(entry.reset) {
			entry = &rateEntry{count: 0, reset: now.Add(time.Minute)}
			rl.items[ip] = entry
		}
		entry.count++
		count := entry.count
		reset := entry.reset
		rl.mu.Unlock()

		if count > rl.limit {
			retry := int(time.Until(reset).Seconds())
			c.Header("Retry-After", fmt.Sprintf("%d", retry))
			utils.RespondError(c, utils.NewAppError(http.StatusTooManyRequests, "RATE_LIMIT", "too many requests", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}
