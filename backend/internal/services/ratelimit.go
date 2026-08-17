package services

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter is the rate-limit abstraction the AI handler depends on so tests can
// substitute an in-memory implementation instead of a live Redis.
type Limiter interface {
	// Allow reports whether the key may proceed given max requests per window.
	Allow(ctx context.Context, scope, key string, max int, window time.Duration) (bool, error)
}

// RedisLimiter implements a sliding-window rate limit with Redis sorted sets.
type RedisLimiter struct{ rdb *redis.Client }

func NewRedisLimiter(rdb *redis.Client) *RedisLimiter { return &RedisLimiter{rdb: rdb} }

// NoopLimiter allows every request; used only when Redis is unreachable so the
// API keeps serving (with a loud startup log) instead of failing hard.
type NoopLimiter struct{}

func (NoopLimiter) Allow(_ context.Context, _ string, _ string, _ int, _ time.Duration) (bool, error) {
	return true, nil
}

func (r *RedisLimiter) Allow(ctx context.Context, scope, key string, max int, window time.Duration) (bool, error) {
	if max <= 0 {
		return true, nil
	}
	redisKey := "ratelimit:" + scope + ":" + key
	now := float64(time.Now().UnixMilli())
	cutoff := now - float64(window.Milliseconds())
	pipe := r.rdb.TxPipeline()
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatFloat(cutoff, 'f', -1, 64))
	pipe.ZAdd(ctx, redisKey, redis.Z{Score: now, Member: now})
	pipe.Expire(ctx, redisKey, window+time.Second)
	pipe.ZCard(ctx, redisKey)
	result, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}
	count := result[len(result)-1].(*redis.IntCmd).Val()
	return count <= int64(max), nil
}
