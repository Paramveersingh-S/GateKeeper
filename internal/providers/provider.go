package providers

import (
	"context"
	"net/http"
)

// Provider defines the interface for an LLM provider adapter.
type Provider interface {
	// Name returns the provider's identifier (e.g., "gemini", "openai")
	Name() string
	
	// RouteRequest routes the incoming HTTP request to the provider.
	// It handles authentication, URL rewriting, and execution.
	RouteRequest(ctx context.Context, req *http.Request, key string) (*http.Response, error)
	
	// SupportStreaming returns true if the provider supports SSE streaming.
	SupportsStreaming() bool
}

// KeyPool manages a set of API keys for a provider.
type KeyPool interface {
	// GetKey returns the next available healthy key.
	GetKey() (string, error)
	
	// MarkFailure marks a key as failed (for circuit breaking).
	MarkFailure(key string)
	
	// MarkSuccess marks a key as successful (resets circuit breaker).
	MarkSuccess(key string)
}
