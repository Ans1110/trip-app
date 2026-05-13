package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/redis/go-redis/v9"
)

var ErrRateLimited = errors.New("rate limit exceeded")

type rateAction string

const (
	rateLogin    rateAction = "login"
	rateRegister rateAction = "register"
	rateForgot   rateAction = "forgot_password"
	rateTOTP     rateAction = "totp"
)

type rateLimiter struct {
	rdb   *redis.Client
	rules map[rateAction]config.RateLimitRule
}

func newRateLimiter(rdb *redis.Client, cfg config.RateLimitConfig) *rateLimiter {
	return &rateLimiter{
		rdb: rdb,
		rules: map[rateAction]config.RateLimitRule{
			rateLogin:    cfg.Login,
			rateRegister: cfg.Register,
			rateForgot:   cfg.ForgotPassword,
			rateTOTP:     cfg.TOTP,
		},
	}
}

// allow increments the counter and returns ErrRateLimited if the caller has
// exceeded the configured limit for the action+identifier. Disabled when redis
// is nil or the rule has zero limit/window.
func (l *rateLimiter) allow(ctx context.Context, action rateAction, identifier string) error {
	if l == nil || l.rdb == nil {
		return nil
	}
	rule, ok := l.rules[action]
	if !ok || rule.Limit <= 0 || rule.Window <= 0 {
		return nil
	}
	if identifier == "" {
		return nil
	}
	key := fmt.Sprintf("auth:rl:%s:%s", action, identifier)
	cnt, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if cnt == 1 {
		if err := l.rdb.Expire(ctx, key, rule.Window).Err(); err != nil {
			return err
		}
	}
	if cnt > int64(rule.Limit) {
		return ErrRateLimited
	}
	return nil
}

// resetWindow lets a successful operation drop its rate-limit counter so
// genuine retries after a successful action don't accumulate.
func (l *rateLimiter) resetWindow(ctx context.Context, action rateAction, identifier string) {
	if l == nil || l.rdb == nil || identifier == "" {
		return
	}
	_ = l.rdb.Del(ctx, fmt.Sprintf("auth:rl:%s:%s", action, identifier)).Err()
}
