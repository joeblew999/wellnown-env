package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// connectToNATS establishes connection to NATS server
func connectToNATS() error {
	natsMu.Lock()
	defer natsMu.Unlock()

	if natsConnected {
		return nil
	}

	url := getNatsURL()
	fmt.Printf("Connecting to NATS at %s...\n", url)

	nc, err := nats.Connect(url,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			fmt.Printf("NATS disconnected: %v\n", err)
			natsMu.Lock()
			natsConnected = false
			natsMu.Unlock()
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			fmt.Println("NATS reconnected!")
			natsMu.Lock()
			natsConnected = true
			natsMu.Unlock()
		}),
	)
	if err != nil {
		return fmt.Errorf("connecting to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("creating JetStream: %w", err)
	}

	// Get or create the KV bucket for theme sync (bucket name from env)
	ctx := context.Background()
	bucketName := getNatsKVBucket()
	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      bucketName,
		Description: "Via configuration for live sync",
		TTL:         0, // No TTL - persist forever
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("creating KV bucket: %w", err)
	}

	natsConn = nc
	natsJS = js
	natsKV = kv
	natsConnected = true

	fmt.Println("Connected to NATS!")

	// Start watching theme changes
	go watchThemeChanges(ctx)

	// Start watching counter changes
	go watchCounterChanges(ctx)

	// Start watching config changes
	go watchConfigChanges(ctx)

	// Subscribe to chat messages
	go subscribeToChatMessages()

	// Start watching services registry changes
	go watchServicesChanges(ctx)

	// Register this via instance in the services registry
	go func() {
		if err := registerViaService(ctx); err != nil {
			fmt.Printf("Failed to register via service: %v\n", err)
		}
	}()

	// Start NATS responder for process status (so VIA can fetch via NATS)
	go func() {
		_ = startProcessStatusResponder(ctx)
	}()

	// Subscribe to process update broadcasts (for /processes-nats)
	go func() {
		_ = startProcessUpdatesSubscription()
	}()

	// NATS control responder for process start/stop/restart
	go func() {
		_ = startProcessControlResponder(ctx)
	}()

	return nil
}
