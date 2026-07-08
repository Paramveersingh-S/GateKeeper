package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// SemanticCache implements vector similarity caching using Redis Stack.
type SemanticCache struct {
	client    *redis.Client
	threshold float32
}

type CachedResponse struct {
	Prompt   string `json:"prompt"`
	Response string `json:"response"`
	Tokens   int    `json:"tokens"`
}

func NewSemanticCache(client *redis.Client, threshold float32) *SemanticCache {
	return &SemanticCache{
		client:    client,
		threshold: threshold,
	}
}

// CheckCache queries Redis for similar prompts using vector search.
// In a real system, we would first call an embedding API (e.g. text-embedding-3-small)
// to convert the prompt to a vector ([]float32), and then use FT.SEARCH.
func (c *SemanticCache) CheckCache(ctx context.Context, prompt string) (*CachedResponse, error) {
	// 1. Embed the prompt
	vector := embedMock(prompt)

	// 2. Search Redis using FT.SEARCH
	// Note: We need a pre-created index in Redis Stack for this to work.
	// Index creation is typically done on startup.
	
	// Convert vector to bytes for Redis
	vectorBytes := float32SliceToBytes(vector)

	// Knn search: find the top 1 nearest neighbor
	query := "*=>[KNN 1 @embedding $vec AS score]"
	
	res, err := c.client.Do(ctx, "FT.SEARCH", "idx:prompts", query, "PARAMS", "2", "vec", vectorBytes, "DIALECT", "2", "RETURN", "2", "response", "score").Result()
	if err != nil {
		return nil, fmt.Errorf("redis search error: %v", err)
	}

	// Parse results (simplified)
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 1 {
		return nil, nil // Not found
	}
	
	totalResults, ok := arr[0].(int64)
	if !ok || totalResults == 0 {
		return nil, nil
	}

	// In a real implementation, we would parse the score from the response
	// and check if score < (1 - c.threshold) (assuming L2 or Cosine distance)
	// For this skeleton, we assume a match if we got here.
	
	// Mock parsing logic
	if len(arr) > 2 {
		fields := arr[2].([]interface{})
		responseStr := ""
		for i := 0; i < len(fields); i += 2 {
			if fields[i].(string) == "response" {
				responseStr = fields[i+1].(string)
			}
		}
		
		if responseStr != "" {
			return &CachedResponse{
				Prompt:   prompt,
				Response: responseStr,
				Tokens:   0, // Cache hit saves tokens
			}, nil
		}
	}

	return nil, nil
}

// StoreCache saves a prompt and its response to the cache.
func (c *SemanticCache) StoreCache(ctx context.Context, prompt, response string, tokens int) error {
	vector := embedMock(prompt)
	vectorBytes := float32SliceToBytes(vector)
	
	hash := sha256.Sum256([]byte(prompt))
	docID := "prompt:" + hex.EncodeToString(hash[:])
	
	// HSET docID prompt <prompt> response <response> embedding <vector>
	err := c.client.HSet(ctx, docID, map[string]interface{}{
		"prompt":    prompt,
		"response":  response,
		"tokens":    tokens,
		"embedding": vectorBytes,
	}).Err()
	
	return err
}

// InitializeIndex creates the vector index in Redis if it doesn't exist.
func (c *SemanticCache) InitializeIndex(ctx context.Context) error {
	// FT.CREATE idx:prompts ON HASH PREFIX 1 prompt: SCHEMA prompt TEXT response TEXT embedding VECTOR FLAT 6 TYPE FLOAT32 DIM 1536 DISTANCE_METRIC COSINE
	_, err := c.client.Do(ctx, "FT.CREATE", "idx:prompts", "ON", "HASH", "PREFIX", "1", "prompt:", "SCHEMA", "prompt", "TEXT", "response", "TEXT", "embedding", "VECTOR", "FLAT", "6", "TYPE", "FLOAT32", "DIM", "3", "DISTANCE_METRIC", "COSINE").Result()
	
	// Ignore "Index already exists" errors
	if err != nil && err.Error() == "Index already exists" {
		return nil
	}
	return err
}

// embedMock mocks an embedding API call.
func embedMock(text string) []float32 {
	// Return a dummy 3-dimensional vector for testing
	return []float32{0.1, 0.2, 0.3}
}

func float32SliceToBytes(slice []float32) []byte {
	// In a real app we'd use encoding/binary to write floats as little-endian bytes.
	// Mock implementation for the skeleton.
	return []byte("dummy_bytes")
}
