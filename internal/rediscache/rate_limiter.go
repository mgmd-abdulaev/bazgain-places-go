package rediscache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var fixedWindowScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local ttl = redis.call('TTL', KEYS[1])
return {current, ttl}
`)

type RateLimiter struct{ client *redis.Client }

func NewRateLimiter(client *redis.Client) *RateLimiter { return &RateLimiter{client: client} }
func (l *RateLimiter) Allow(ctx context.Context, scope, clientIP string, limit int64, window time.Duration) (bool, time.Duration, error) {
	windowSeconds := int64(window / time.Second)
	if windowSeconds < 1 {
		return false, 0, fmt.Errorf("rate limit window must be at least one second")
	}
	now := time.Now().UTC().Unix()
	bucket := now / windowSeconds
	ttlToBoundary := windowSeconds - now%windowSeconds
	key := fmt.Sprintf("places-go:ratelimit:%s:%s:%d", scope, clientIP, bucket)
	value, err := fixedWindowScript.Run(ctx, l.client, []string{key}, ttlToBoundary).Slice()
	if err != nil {
		return false, 0, err
	}
	if len(value) != 2 {
		return false, 0, fmt.Errorf("unexpected rate limit result")
	}
	count, ok := value[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected rate limit count")
	}
	ttl, ok := value[1].(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected rate limit ttl")
	}
	return count <= limit, time.Duration(ttl) * time.Second, nil
}

func (l *RateLimiter) SessionRevoked(ctx context.Context, sessionID string) (bool, error) {
	value, err := l.client.Exists(ctx, "auth:session:revoked:"+sessionID).Result()
	return value > 0, err
}
