package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Paramveersingh-S/GateKeeper/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

func main() {
	// Initialize PostgreSQL Client
	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, "postgres://gatekeeper:password@127.0.0.1:5432/gatekeeper?sslmode=disable")
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer dbpool.Close()

	natsURL := "nats://127.0.0.1:4222" // Using 127.0.0.1 avoids IPv6 Docker binding issues on Linux
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to get JetStream context: %v", err)
	}

	// Ensure stream exists before subscribing
	streamName := "USAGE_EVENTS"
	_, err = js.StreamInfo(streamName)
	if err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"usage.>"},
		})
		if err != nil {
			log.Printf("Warning: failed to create stream: %v", err)
		}
	}

	log.Println("Starting Usage Consumer...")

	// Subscribe to usage events
	sub, err := js.Subscribe("usage.log", func(msg *nats.Msg) {
		var event telemetry.UsageEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Error decoding message: %v", err)
			msg.Nak()
			return
		}

		// Insert into PostgreSQL
		_, err = dbpool.Exec(context.Background(), 
			`INSERT INTO usage_events (tenant_id, model, provider, prompt_tokens, completion_tokens, cost_usd, cache_hit, latency_ms, status) 
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'success')`,
			event.TenantID, event.Model, event.Provider, event.PromptTokens, event.CompletionTokens, 
			event.CostUSD, event.CacheHit, event.LatencyMs)
		
		if err != nil {
			log.Printf("Failed to insert usage event: %v", err)
			// Wait to Ack so it gets retried or sent to DLQ
			msg.Nak()
			return
		}

		log.Printf("Saved usage event for %s, cost: $%.5f", event.TenantID, event.CostUSD)
		msg.Ack()
	}, nats.Durable("USAGE_CONSUMER"), nats.ManualAck())

	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	
	log.Println("Shutting down Usage Consumer...")
	sub.Unsubscribe()
}
