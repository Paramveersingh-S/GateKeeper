package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter implements the Limiter interface using Redis.
type RedisLimiter struct {
	client *redis.Client
}

// NewRedisLimiter creates a new Redis-backed rate limiter.
func NewRedisLimiter(client *redis.Client) *RedisLimiter {
	return &RedisLimiter{
		client: client,
	}
}

func (l *RedisLimiter) AllowRequest(ctx context.Context, key string, policy Policy) (Result, error) {
	now := time.Now()
	nowSec := now.Unix()
	nowMs := now.UnixMilli()

	switch policy.Algorithm {
	case FixedWindow:
		windowKey := fmt.Sprintf("rl:fw:%s:%d", key, nowSec/60)
		res, err := FixedWindowScript.Run(ctx, l.client, []string{windowKey}, policy.RPM, 1, 60).Result()
		return parseResult(res, err)

	case SlidingWindowLog:
		windowKey := fmt.Sprintf("rl:swl:%s", key)
		windowStart := nowMs - 60000
		res, err := SlidingWindowLogScript.Run(ctx, l.client, []string{windowKey}, policy.RPM, 1, nowMs, windowStart, 60).Result()
		return parseResult(res, err)

	case SlidingWindowCounter:
		currMin := nowSec / 60
		prevMin := currMin - 1
		currKey := fmt.Sprintf("rl:swc:%s:%d", key, currMin)
		prevKey := fmt.Sprintf("rl:swc:%s:%d", key, prevMin)
		weight := 1.0 - float64(now.Second())/60.0

		res, err := SlidingWindowCounterScript.Run(ctx, l.client, []string{currKey, prevKey}, policy.RPM, 1, weight, 60).Result()
		return parseResult(res, err)

	case TokenBucket:
		fallthrough
	default:
		bucketKey := fmt.Sprintf("rl:tb:%s", key)
		rate := float64(policy.RPM) / 60.0 // tokens per second
		if rate <= 0 {
			rate = 1
		}
		res, err := TokenBucketScript.Run(ctx, l.client, []string{bucketKey}, policy.RPM, rate, 1, nowSec).Result()
		return parseResult(res, err)
	}
}

func (l *RedisLimiter) ReserveTokens(ctx context.Context, key string, tokens int, policy Policy) (Result, error) {
	bucketKey := fmt.Sprintf("rl:tbtokens:%s", key)
	rate := float64(policy.TPM) / 60.0
	if rate <= 0 {
		rate = 100 // default reasonable rate if not configured
	}
	capacity := policy.TPM
	nowSec := time.Now().Unix()

	res, err := ReserveReconcileScript.Run(ctx, l.client, []string{bucketKey}, "reserve", capacity, rate, tokens, nowSec).Result()
	return parseResult(res, err)
}

func (l *RedisLimiter) ReconcileTokens(ctx context.Context, key string, estimatedTokens, actualTokens int, policy Policy) error {
	if estimatedTokens <= actualTokens {
		// We didn't over-reserve, nothing to refund
		return nil
	}

	diff := estimatedTokens - actualTokens
	bucketKey := fmt.Sprintf("rl:tbtokens:%s", key)
	
	_, err := ReserveReconcileScript.Run(ctx, l.client, []string{bucketKey}, "reconcile", diff).Result()
	return err
}

func parseResult(res interface{}, err error) (Result, error) {
	if err != nil {
		return Result{}, err
	}

	vals, ok := res.([]interface{})
	if !ok || len(vals) < 2 {
		return Result{}, fmt.Errorf("unexpected script result format")
	}

	allowed := vals[0].(int64) == 1
	remaining := int(vals[1].(int64))

	return Result{
		Allowed:   allowed,
		Remaining: remaining,
	}, nil
}
