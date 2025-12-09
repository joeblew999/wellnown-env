// Example: Embedded NATS JetStream with standalone/leaf node topology
//
// This demonstrates the core architecture:
// - Every binary embeds a NATS server
// - No NATS_HUB env → standalone mode (dev)
// - NATS_HUB set → leaf node connecting to hub (prod)
//
// Run standalone:
//   go run main.go
//
// Run as hub (for other services to connect to):
//   NATS_PORT=4222 NATS_NAME=hub go run main.go
//
// Run as leaf node:
//   NATS_HUB=nats://localhost:4222 NATS_PORT=4223 NATS_NAME=svc-a go run main.go
//
// Use process-compose to run multiple instances - see process-compose.yaml
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Configuration from environment
	name := getEnv("NATS_NAME", "node-"+uuid.New().String()[:8])
	port := getEnvInt("NATS_PORT", 0) // 0 = random port
	hubURL := os.Getenv("NATS_HUB")   // empty = standalone
	dataDir := os.Getenv("NATS_DATA") // empty = in-memory

	fmt.Printf("Starting NATS node: %s\n", name)
	fmt.Printf("  Port: %d (0 = random)\n", port)
	fmt.Printf("  Hub:  %s (empty = standalone)\n", hubURL)
	fmt.Printf("  Data: %s (empty = in-memory)\n", dataDir)
	fmt.Println()

	// Configure NATS server options
	opts := &server.Options{
		ServerName: name,
		Port:       port,
		JetStream:  true,
		StoreDir:   dataDir,
		// Disable logging noise for demo
		NoLog:  false,
		Debug:  false,
		Trace:  false,
	}

	// If hub URL provided, configure as leaf node
	if hubURL != "" {
		u, err := url.Parse(hubURL)
		if err != nil {
			return fmt.Errorf("parsing hub URL: %w", err)
		}
		opts.LeafNode = server.LeafNodeOpts{
			Remotes: []*server.RemoteLeafOpts{
				{URLs: []*url.URL{u}},
			},
		}
		fmt.Printf("Configured as LEAF NODE connecting to: %s\n", hubURL)
	} else {
		// Enable leaf node listening so other nodes can connect
		opts.LeafNode = server.LeafNodeOpts{
			Port: port + 1000, // Leaf port = client port + 1000
		}
		fmt.Printf("Configured as STANDALONE (leaf listen port: %d)\n", port+1000)
	}

	// Create and start the embedded server
	ns, err := server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	// Start in background
	go ns.Start()

	// Wait for server to be ready (15s for go run which includes compile time)
	if !ns.ReadyForConnections(15 * time.Second) {
		return fmt.Errorf("server not ready")
	}

	fmt.Printf("\nNATS server ready!\n")
	fmt.Printf("  Client URL: %s\n", ns.ClientURL())
	fmt.Printf("  Cluster:    %v\n", ns.ClusterAddr())
	fmt.Println()

	// Connect as a client to our own embedded server
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		return fmt.Errorf("connecting to server: %w", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("creating jetstream: %w", err)
	}

	ctx := context.Background()

	// Create or get the services_registry KV bucket
	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      "services_registry",
		Description: "Service registration for wellnown-env",
		TTL:         30 * time.Second, // Entries expire if not refreshed
	})
	if err != nil {
		return fmt.Errorf("creating KV bucket: %w", err)
	}

	fmt.Printf("KV bucket 'services_registry' ready\n\n")

	// Register this service
	key := fmt.Sprintf("%s.%s", "demo", name)
	registration := fmt.Sprintf(`{"name":"%s","host":"%s","time":"%s"}`,
		name, ns.ClientURL(), time.Now().Format(time.RFC3339))

	rev, err := kv.Put(ctx, key, []byte(registration))
	if err != nil {
		return fmt.Errorf("registering service: %w", err)
	}
	fmt.Printf("Registered: %s (rev %d)\n", key, rev)

	// Start heartbeat to keep registration alive
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			registration := fmt.Sprintf(`{"name":"%s","host":"%s","time":"%s"}`,
				name, ns.ClientURL(), time.Now().Format(time.RFC3339))
			if _, err := kv.Put(ctx, key, []byte(registration)); err != nil {
				fmt.Printf("Heartbeat failed: %v\n", err)
			} else {
				fmt.Printf("Heartbeat: %s\n", key)
			}
		}
	}()

	// Watch for other services
	watcher, err := kv.WatchAll(ctx)
	if err != nil {
		return fmt.Errorf("watching KV: %w", err)
	}

	go func() {
		for entry := range watcher.Updates() {
			if entry == nil {
				continue
			}
			op := "PUT"
			if entry.Operation() == jetstream.KeyValueDelete {
				op = "DEL"
			}
			fmt.Printf("[WATCH] %s %s = %s\n", op, entry.Key(), string(entry.Value()))
		}
	}()

	// List all registered services periodically
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Println("\n--- Registered Services ---")
			keys, err := kv.Keys(ctx)
			if err != nil {
				fmt.Printf("Error listing keys: %v\n", err)
				continue
			}
			for _, k := range keys {
				entry, err := kv.Get(ctx, k)
				if err != nil {
					continue
				}
				fmt.Printf("  %s: %s\n", k, string(entry.Value()))
			}
			fmt.Println("---------------------------\n")
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")

	// Deregister
	if err := kv.Delete(ctx, key); err != nil {
		fmt.Printf("Failed to deregister: %v\n", err)
	} else {
		fmt.Printf("Deregistered: %s\n", key)
	}

	// Stop watcher
	watcher.Stop()

	// Shutdown server
	ns.Shutdown()
	ns.WaitForShutdown()

	fmt.Println("Goodbye!")
	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
