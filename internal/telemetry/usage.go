package telemetry

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type UsageEvent struct {
	TenantID         string    `json:"tenant_id"`
	Model            string    `json:"model"`
	Provider         string    `json:"provider"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CostUSD          float64   `json:"cost_usd"`
	CacheHit         bool      `json:"cache_hit"`
	LatencyMs        int64     `json:"latency_ms"`
	Timestamp        time.Time `json:"timestamp"`
}

type UsageProducer struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewUsageProducer(natsURL string) (*UsageProducer, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	
	// Ensure the stream exists
	streamName := "USAGE_EVENTS"
	_, err = js.StreamInfo(streamName)
	if err != nil {
		// Create stream if not exists
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"usage.>"},
		})
		if err != nil {
			log.Printf("Warning: failed to create stream: %v", err)
		}
	}

	return &UsageProducer{
		nc: nc,
		js: js,
	}, nil
}

func (p *UsageProducer) Publish(event UsageEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	
	// Async publish to not block the hot path
	_, err = p.js.PublishAsync("usage.log", data)
	return err
}

func (p *UsageProducer) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}
