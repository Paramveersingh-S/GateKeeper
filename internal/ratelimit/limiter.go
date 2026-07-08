package ratelimit

import (
	"context"
	"time"
)

// Policy defines the rate limiting rules for a tenant or API key.
type Policy struct {
	// Algorithm specifies which rate limit algorithm to use
	Algorithm Algorithm
	
	// RPM is Requests Per Minute
	RPM int
	
	// TPM is Tokens Per Minute
	TPM int
	
	// TPD is Tokens Per Day
	TPD int
	
	// MaxConcurrent is the maximum number of concurrent requests
	MaxConcurrent int
}

// Algorithm represents the type of rate limiting strategy.
type Algorithm string

const (
	TokenBucket          Algorithm = "token_bucket"
	SlidingWindowCounter Algorithm = "sliding_window_counter"
	SlidingWindowLog     Algorithm = "sliding_window_log"
	FixedWindow          Algorithm = "fixed_window"
)

// Result is returned by the rate limiter to indicate if the request is allowed.
type Result struct {
	Allowed   bool
	Remaining int
	RetryAfter time.Duration
	// Optionally return token-specific info
	TokensRemaining int
}

// Limiter is the interface that all rate limiting algorithms must implement.
type Limiter interface {
	// AllowRequest checks if a request is allowed based on request count (RPM).
	AllowRequest(ctx context.Context, key string, policy Policy) (Result, error)
	
	// ReserveTokens attempts to reserve a specified number of tokens.
	ReserveTokens(ctx context.Context, key string, tokens int, policy Policy) (Result, error)
	
	// ReconcileTokens adjusts the token usage after the actual cost is known.
	// Used primarily for streaming responses where exact token count is not known upfront.
	ReconcileTokens(ctx context.Context, key string, estimatedTokens, actualTokens int, policy Policy) error
}
