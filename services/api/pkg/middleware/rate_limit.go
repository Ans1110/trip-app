package middleware

import (
	"fmt"
	"time"

	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func RateLimit(rdb *redis.Client, maxRequests int, window time.Duration, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rate_limit:%s:%s", c.ClientIP(), c.FullPath())
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			logger.Error("Error occurred while incrementing rate limit counter")
			c.Next()
			return
		}

		if count == 1 {
			if err := rdb.Expire(ctx, key, window).Err(); err != nil {
				logger.Error("Error occurred while setting expiration for rate limit key")
				c.Next()
				return
			}
		}

		if count > int64(maxRequests) {
			response.TooManyRequests(c)
			c.Abort()
			return
		}
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, maxRequests-int(count))))
		c.Next()
	}
}
