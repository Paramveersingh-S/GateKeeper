package providers

import (
	"errors"
	"sync"
	"time"
)

var ErrNoHealthyKeys = errors.New("no healthy API keys available")

type keyStatus struct {
	key           string
	failures      int
	lastFailure   time.Time
	circuitOpen   bool
}

// SimplePool implements KeyPool with round-robin and basic circuit breaking.
type SimplePool struct {
	mu           sync.Mutex
	keys         []*keyStatus
	currentIndex int
	
	// Configurable thresholds
	MaxFailures   int
	ResetTimeout  time.Duration
}

func NewSimplePool(apiKeys []string) *SimplePool {
	pool := &SimplePool{
		keys:         make([]*keyStatus, len(apiKeys)),
		MaxFailures:  3,
		ResetTimeout: 30 * time.Second,
	}
	
	for i, k := range apiKeys {
		pool.keys[i] = &keyStatus{key: k}
	}
	
	return pool
}

func (p *SimplePool) GetKey() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if len(p.keys) == 0 {
		return "", ErrNoHealthyKeys
	}
	
	startIdx := p.currentIndex
	now := time.Now()
	
	for {
		status := p.keys[p.currentIndex]
		
		// Check circuit breaker
		if status.circuitOpen {
			if now.Sub(status.lastFailure) > p.ResetTimeout {
				// Half-open: try again
				status.circuitOpen = false
			}
		}
		
		p.currentIndex = (p.currentIndex + 1) % len(p.keys)
		
		if !status.circuitOpen {
			return status.key, nil
		}
		
		if p.currentIndex == startIdx {
			// Checked all keys, none are healthy
			return "", ErrNoHealthyKeys
		}
	}
}

func (p *SimplePool) MarkFailure(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for _, status := range p.keys {
		if status.key == key {
			status.failures++
			status.lastFailure = time.Now()
			if status.failures >= p.MaxFailures {
				status.circuitOpen = true
			}
			break
		}
	}
}

func (p *SimplePool) MarkSuccess(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for _, status := range p.keys {
		if status.key == key {
			status.failures = 0
			status.circuitOpen = false
			break
		}
	}
}
