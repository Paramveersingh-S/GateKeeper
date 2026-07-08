package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/Paramveersingh-S/GateKeeper/internal/ratelimit"
	"github.com/redis/go-redis/v9"
)

type Gateway struct {
	limiter ratelimit.Limiter
	mockLLM *url.URL
}

func main() {
	// Initialize Redis Client
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	// Create Limiter
	limiter := ratelimit.NewRedisLimiter(rdb)

	// URL of the mock LLM backend (defined in docker-compose.yml)
	mockLLMURL, _ := url.Parse("http://localhost:8081")

	gw := &Gateway{
		limiter: limiter,
		mockLLM: mockLLMURL,
	}

	mux := http.NewServeMux()
	
	// Mock proxy handler for OpenAI compatible endpoints
	proxy := httputil.NewSingleHostReverseProxy(gw.mockLLM)
	mux.Handle("/v1/chat/completions", gw.rateLimitMiddleware(proxy))
	mux.Handle("/v1/embeddings", gw.rateLimitMiddleware(proxy))

	log.Println("Starting GateKeeper on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// rateLimitMiddleware applies token bucket rate limiting and token reservation.
func (gw *Gateway) rateLimitMiddleware(next http.Handler) http.Handler {
	tokenizer := ratelimit.NewHeuristicTokenizer()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		tenantID := "tenant_mock"
		if apiKey != "" {
			tenantID = "tenant_" + apiKey
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		policy := ratelimit.Policy{
			Algorithm: ratelimit.TokenBucket,
			RPM:       100,
			TPM:       10000, // 10k tokens per minute
		}

		// 1. RPM Check
		res, err := gw.limiter.AllowRequest(ctx, tenantID, policy)
		if err != nil || !res.Allowed {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests (RPM)", http.StatusTooManyRequests)
			return
		}

		// 2. Token Estimation
		estimatedTokens := 50 // baseline
		// In a real app we'd parse the JSON body to extract the prompt string for estimation
		// e.g. estimatedTokens = tokenizer.EstimateTokens(parsedPrompt)

		// 3. Reserve Tokens
		tokenRes, err := gw.limiter.ReserveTokens(ctx, tenantID, estimatedTokens, policy)
		if err != nil || !tokenRes.Allowed {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Tokens (TPM)", http.StatusTooManyRequests)
			return
		}

		// 4. Wrap ResponseWriter to capture actual usage from response (for streaming/reconciliation)
		// Usually this involves parsing the final SSE chunk or response body for usage metadata.
		// For demo purposes, we will mock the actual tokens used as slightly less than estimated.
		actualTokens := estimatedTokens - 10 
		if actualTokens < 0 {
			actualTokens = 0
		}

		// Proceed to proxy
		next.ServeHTTP(w, r)

		// 5. Reconcile Tokens asynchronously
		go func(tID string, est, act int) {
			reconcileCtx, recCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer recCancel()
			_ = gw.limiter.ReconcileTokens(reconcileCtx, tID, est, act, policy)
		}(tenantID, estimatedTokens, actualTokens)
	})
}
