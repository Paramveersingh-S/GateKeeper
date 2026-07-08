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

// rateLimitMiddleware applies token bucket rate limiting based on a mock tenant policy.
func (gw *Gateway) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In a real system, we would extract the API key from Authorization header
		// and lookup the tenant policy in Postgres.
		// For now, we mock it.
		apiKey := r.Header.Get("Authorization")
		tenantID := "tenant_mock"
		if apiKey != "" {
			tenantID = "tenant_" + apiKey
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// A mock policy: 100 requests per minute
		policy := ratelimit.Policy{
			Algorithm: ratelimit.TokenBucket,
			RPM:       100,
		}

		res, err := gw.limiter.AllowRequest(ctx, tenantID, policy)
		if err != nil {
			log.Printf("Rate limit check failed: %v", err)
			// Fail open or fail closed? We choose fail closed for safety, but can be configured.
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !res.Allowed {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// Since it's allowed, proceed to the proxy
		next.ServeHTTP(w, r)
	})
}
