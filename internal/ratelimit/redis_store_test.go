package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// setupRedis creates a Redis client for testing. 
// Assumes Redis is running at localhost:6379 (via docker-compose)
func setupRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
	}
	
	// Clean up before test
	client.FlushDB(ctx)
	
	return client
}

func TestRateLimiter_TokenBucket(t *testing.T) {
	client := setupRedis(t)
	defer client.Close()
	
	limiter := NewRedisLimiter(client)
	ctx := context.Background()
	key := "test_tenant"
	
	policy := Policy{
		Algorithm: TokenBucket,
		RPM:       5, // 5 requests per minute -> ~1 token every 12 seconds
	}
	
	// Should allow 5 requests immediately
	for i := 0; i < 5; i++ {
		res, err := limiter.AllowRequest(ctx, key, policy)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !res.Allowed {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}
	
	// 6th request should fail
	res, err := limiter.AllowRequest(ctx, key, policy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res.Allowed {
		t.Error("Expected 6th request to be denied")
	}
}

func TestRateLimiter_Concurrency(t *testing.T) {
	client := setupRedis(t)
	defer client.Close()
	
	limiter := NewRedisLimiter(client)
	ctx := context.Background()
	key := "test_concurrent"
	
	policy := Policy{
		Algorithm: FixedWindow,
		RPM:       10, // 10 requests total
	}
	
	// Fire 50 concurrent requests
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0
	
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, _ := limiter.AllowRequest(ctx, key, policy)
			if res.Allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}
	
	wg.Wait()
	
	if allowedCount != 10 {
		t.Errorf("Expected exactly 10 requests to be allowed under concurrency, got %d", allowedCount)
	}
}

func TestReserveReconcile(t *testing.T) {
	client := setupRedis(t)
	defer client.Close()
	
	limiter := NewRedisLimiter(client)
	ctx := context.Background()
	key := "test_tokens"
	
	policy := Policy{
		TPM: 1000,
	}
	
	// Reserve 800 tokens
	res, err := limiter.ReserveTokens(ctx, key, 800, policy)
	if err != nil || !res.Allowed {
		t.Fatalf("Failed to reserve 800 tokens: %v", err)
	}
	
	// Try to reserve 300 more -> should fail
	res, err = limiter.ReserveTokens(ctx, key, 300, policy)
	if err != nil {
		t.Fatalf("Error on second reserve: %v", err)
	}
	if res.Allowed {
		t.Error("Should not have allowed second reservation exceeding TPM")
	}
	
	// Reconcile: we only used 500 out of 800
	err = limiter.ReconcileTokens(ctx, key, 800, 500, policy)
	if err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}
	
	// Try to reserve 300 more -> should now succeed because 300 were returned
	res, err = limiter.ReserveTokens(ctx, key, 300, policy)
	if err != nil {
		t.Fatalf("Error on third reserve: %v", err)
	}
	if !res.Allowed {
		t.Error("Should have allowed reservation after reconcile")
	}
}
