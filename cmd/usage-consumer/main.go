package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Paramveersingh-S/GateKeeper/internal/telemetry"
	"github.com/nats-io/nats.go"
)

func main() {
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

		// In a real system, we would batch these and write to PostgreSQL.
		// For this skeleton, we just log it.
		log.Printf("Processed usage event for %s, model: %s, cost: $%.5f", 
			event.TenantID, event.Model, event.CostUSD)

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
