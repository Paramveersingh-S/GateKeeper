package providers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeminiAdapter implements the Provider interface for Google Gemini.
type GeminiAdapter struct {
	pool       KeyPool
	httpClient *http.Client
}

func NewGeminiAdapter(keys []string) *GeminiAdapter {
	return &GeminiAdapter{
		pool:       NewSimplePool(keys),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *GeminiAdapter) Name() string {
	return "gemini"
}

func (g *GeminiAdapter) RouteRequest(ctx context.Context, req *http.Request, key string) (*http.Response, error) {
	// Attempt to get a healthy key from the pool
	apiKey, err := g.pool.GetKey()
	if err != nil {
		return nil, fmt.Errorf("gemini adapter: %w", err)
	}

	// For a real gateway, we would translate the OpenAI request format to Gemini format here.
	// For simplicity in this skeleton, we assume the request is already in Gemini format
	// or we just pass it through to the Gemini API endpoint.
	
	// Example endpoint construction
	// https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-pro:generateContent?key=YOUR_API_KEY
	
	// Read the body so we can retry if needed
	bodyBytes, _ := io.ReadAll(req.Body)
	req.Body.Close()

	targetURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta%s?key=%s", req.URL.Path, apiKey)
	
	outReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	
	// Copy headers
	for k, vv := range req.Header {
		for _, v := range vv {
			outReq.Header.Add(k, v)
		}
	}

	resp, err := g.httpClient.Do(outReq)
	if err != nil {
		g.pool.MarkFailure(apiKey)
		return nil, err
	}
	
	// If the status is a 429 or 5xx, we might want to mark failure
	if resp.StatusCode >= 500 || resp.StatusCode == 429 {
		g.pool.MarkFailure(apiKey)
	} else {
		g.pool.MarkSuccess(apiKey)
	}

	return resp, nil
}

func (g *GeminiAdapter) SupportsStreaming() bool {
	return true
}
