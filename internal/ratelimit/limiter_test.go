package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestRedisLimiter_Allow(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()

	limiter := NewRedisLimiter(rdb)
	ctx := context.Background()
	tenantID := "test_tenant_1"

	// Test 1: Basic allowance within limit
	res, err := limiter.ReserveTokens(ctx, tenantID, 5, Policy{TPM: 10}) // cost 5, limit 10
	require.NoError(t, err)
	assert.True(t, res.Allowed, "should allow 5 tokens when limit is 10")

	// Test 2: Allowance that exceeds limit
	res, err = limiter.ReserveTokens(ctx, tenantID, 6, Policy{TPM: 10}) // cost 6, remaining 5 -> should fail
	require.NoError(t, err)
	assert.False(t, res.Allowed, "should reject 6 tokens when only 5 remaining")

	// Test 3: Exact limit
	res, err = limiter.ReserveTokens(ctx, tenantID, 5, Policy{TPM: 10}) // cost 5, remaining 5 -> should pass
	require.NoError(t, err)
	assert.True(t, res.Allowed, "should allow exact remaining 5 tokens")

	// Test 4: Completely exhausted
	res, err = limiter.ReserveTokens(ctx, tenantID, 1, Policy{TPM: 10})
	require.NoError(t, err)
	assert.False(t, res.Allowed, "should reject 1 token when exhausted")

	// Test 5: Wait for refill (fast-forward time in miniredis)
	mr.FastForward(2 * time.Minute) // Expiry is 1 minute, so this resets the window
	
	res, err = limiter.ReserveTokens(ctx, tenantID, 10, Policy{TPM: 10})
	require.NoError(t, err)
	assert.True(t, res.Allowed, "should allow 10 tokens after window reset")
}

func TestRedisLimiter_Concurrency(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()

	limiter := NewRedisLimiter(rdb)
	ctx := context.Background()
	tenantID := "test_tenant_concurrent"
	
	limit := 100
	costPerReq := 1
	
	// Launch 200 goroutines trying to consume 1 token each simultaneously
	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := limiter.ReserveTokens(ctx, tenantID, costPerReq, Policy{TPM: limit})
			if err == nil && res.Allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Exactly 100 requests should have succeeded, no more, no less.
	assert.Equal(t, 100, allowedCount, "exactly 100 requests should be allowed under high concurrency")
}
