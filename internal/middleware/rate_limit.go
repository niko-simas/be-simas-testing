package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitMiddleware struct {
	redis *redis.Client
}

func NewRateLimitMiddleware(redisClient *redis.Client) *RateLimitMiddleware {
	return &RateLimitMiddleware{redis: redisClient}
}

type rateLimitRule struct {
	Limit  int
	Window time.Duration
}

var rules = map[string]rateLimitRule{
	"register":    {Limit: 5, Window: 15 * time.Minute},
	"verify_otp":  {Limit: 10, Window: 10 * time.Minute},
	"resend_otp":  {Limit: 3, Window: 10 * time.Minute},
	"login":       {Limit: 10, Window: 15 * time.Minute},
	"oauth":       {Limit: 10, Window: 15 * time.Minute},
	"refresh":     {Limit: 20, Window: 15 * time.Minute},
}

// Limit applies rate limiting by IP + action name
func (m *RateLimitMiddleware) Limit(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.redis == nil {
			c.Next()
			return
		}

		rule, ok := rules[action]
		if !ok {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", action, c.ClientIP())
		ctx := context.Background()

		count, err := m.redis.Incr(ctx, key).Result()
		if err != nil {
			c.Next() // Redis error — allow request through
			return
		}

		if count == 1 {
			m.redis.Expire(ctx, key, rule.Window)
		}

		if count > int64(rule.Limit) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    "rate_limit_exceeded",
				"message": "Terlalu banyak percobaan, coba lagi nanti",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
