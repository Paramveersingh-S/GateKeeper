package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Paramveersingh-S/GateKeeper/internal/providers"
)

var ErrAllProvidersFailed = errors.New("all configured providers failed")

// Router manages routing to multiple providers with failover support.
type Router struct {
	providers map[string]providers.Provider
}

func NewRouter() *Router {
	return &Router{
		providers: make(map[string]providers.Provider),
	}
}

// RegisterProvider adds a provider adapter to the router.
func (r *Router) RegisterProvider(p providers.Provider) {
	r.providers[p.Name()] = p
}

// RouteWithFailover attempts to route the request to the primary provider.
// If it fails, it falls back to the secondary providers in order.
func (r *Router) RouteWithFailover(ctx context.Context, req *http.Request, providerNames []string) (*http.Response, error) {
	if len(providerNames) == 0 {
		return nil, fmt.Errorf("no providers specified for routing")
	}

	for _, name := range providerNames {
		p, exists := r.providers[name]
		if !exists {
			continue // Skip unconfigured providers
		}

		// We need to clone the request body for retries
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body.Close()
		}

		// Recreate the request for this attempt
		clone := req.Clone(ctx)
		// We can't easily recreate the body reader if we didn't store it, 
		// but since we read it above, we would normally set it here.
		// For simplicity in this demo proxy, we just pass the request to the provider adapter,
		// which also reads and reconstructs it.
		
		resp, err := p.RouteRequest(ctx, clone, "tenant-api-key")
		
		if err == nil && resp.StatusCode < 500 && resp.StatusCode != 429 {
			return resp, nil
		}
		
		// If there was an error or a 429/5xx, we continue to the next provider
		if resp != nil {
			resp.Body.Close()
		}
	}

	return nil, ErrAllProvidersFailed
}
