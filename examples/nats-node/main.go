// nats-node: Embedded NATS JetStream server (hub or leaf node)
//
// This is the NATS infrastructure binary - it MUST NOT go down.
// Run separately from application services for resilience.
//
// Modes:
//   - Standalone (hub): No NATS_HUB env - acts as the central hub
//   - Leaf node: NATS_HUB set - connects to hub cluster
//
// IMPORTANT: Offline-First Architecture
//
// When running as a leaf node, your service embeds its own NATS server that:
//   - Works completely offline when hub is unavailable
//   - Automatically syncs with hub when connectivity is restored
//   - Persists data locally (if NATS_DATA is set)
//
// This enables building servers that work in disconnected environments
// (field devices, edge nodes, mobile units) and sync when back online.
//
// To embed in your own Go service, import the nats-server package:
//
//   import "github.com/nats-io/nats-server/v2/server"
//
//   opts := &server.Options{
//       JetStream: true,
//       StoreDir:  "./data",
//       LeafNode: server.LeafNodeOpts{
//           Remotes: []*server.RemoteLeafOpts{{URLs: hubURLs}},
//       },
//   }
//   ns, _ := server.NewServer(opts)
//   ns.Start()
//
// DESIGN PRINCIPLE: Code Over Configuration
//
// Common functionality that every NATS hub/leaf needs is implemented HERE
// in Go, not in Taskfiles or process-compose.yaml. This keeps config files
// minimal and behavior consistent. Examples:
//   - Service self-registration with heartbeat
//   - KV bucket creation (services_registry)
//   - Process-compose polling and publishing
//   - Graceful shutdown with deregistration
//
// As patterns emerge, we add them here so all nodes get them automatically.
//
// Features:
//   - Embedded NATS JetStream server
//   - Service registry KV bucket (services_registry)
//   - Process-compose polling → publishes to pc.processes.updates
//   - Automatic heartbeat for service registration
//
// Run as hub:
//   NATS_PORT=4222 NATS_NAME=hub go run main.go
//
// Run as leaf node:
//   NATS_HUB=nats://localhost:4222 NATS_PORT=4223 NATS_NAME=svc-a go run main.go
//
// Environment:
//   NATS_NAME  - Node name (default: random UUID)
//   NATS_PORT  - Client port (default: random)
//   NATS_HUB   - Hub URL for leaf mode (empty = standalone)
//   NATS_DATA  - Data directory (empty = in-memory)
//   NATS_AUTH  - Auth mode: none, token, nkey, jwt (default: none)
//   PC_URL     - Process-compose API URL (default: http://localhost:8181)
//
// Auth files in .auth/ directory (see auth.go for details):
//   .auth/mode         - Current auth mode
//   .auth/token        - Shared token (token mode)
//   .auth/user.pub     - NKey public key (nkey mode)
//   .auth/user.nk      - NKey seed (client-side, nkey mode)
//   .auth/creds/       - JWT credentials (jwt mode)
//
// See process-compose.yaml for orchestrated multi-node setup
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joeblew999/wellnown-env/pkg/env"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ProcessState represents a single process from process-compose API
type ProcessState struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	IsRunning bool   `json:"is_running"`
	Pid       int    `json:"pid"`
	Health    string `json:"health,omitempty"`
	Restarts  int    `json:"restarts"`
}

// ProcessStates wraps the API response
type ProcessStates struct {
	States []ProcessState `json:"data"`
}

const processUpdatesSubject = "pc.processes.updates"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Resolve any ref+ secrets in environment variables (vals)
	// This allows secrets to come from Vault, 1Password, files, etc.
	// For local dev, use ref+file://./secrets/mykey.txt
	if env.HasSecretRefs() {
		fmt.Println("Resolving secrets...")
		if err := env.ResolveEnvSecrets(); err != nil {
			return fmt.Errorf("resolving secrets: %w", err)
		}
		fmt.Printf("  Resolved: %v\n", env.ListSecretRefs())
	}

	// Configuration from environment
	name := env.GetEnv("NATS_NAME", "node-"+uuid.New().String()[:8])
	port := env.GetEnvInt("NATS_PORT", 0) // 0 = random port
	hubURL := os.Getenv("NATS_HUB")   // empty = standalone
	dataDir := os.Getenv("NATS_DATA") // empty = in-memory

	// Load auth configuration from .auth/ directory
	authCfg, err := LoadAuthConfig()
	if err != nil {
		return fmt.Errorf("loading auth config: %w", err)
	}

	fmt.Printf("Starting NATS node: %s\n", name)
	fmt.Printf("  Port: %d (0 = random)\n", port)
	fmt.Printf("  Hub:  %s (empty = standalone)\n", hubURL)
	fmt.Printf("  Data: %s (empty = in-memory)\n", dataDir)
	fmt.Printf("  Auth: %s\n", authCfg.Mode)
	fmt.Println()

	// Configure NATS server options
	opts := &server.Options{
		ServerName: name,
		Port:       port,
		JetStream:  true,
		StoreDir:   dataDir,
		// Disable logging noise for demo
		NoLog: false,
		Debug: false,
		Trace: false,
	}

	// Configure authentication based on lifecycle phase
	if err := ConfigureAuth(opts, authCfg); err != nil {
		return fmt.Errorf("configuring auth: %w", err)
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

	// Connect as a client to our own embedded server with appropriate auth
	clientOpts, err := GetClientConnectOptions(authCfg)
	if err != nil {
		return fmt.Errorf("getting client auth options: %w", err)
	}
	nc, err := nats.Connect(ns.ClientURL(), clientOpts...)
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

	// Start process-compose poller - publishes to pc.processes.updates
	go startProcessComposePoller(nc)

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


// fetchProcessStates calls process-compose API to get process states
func fetchProcessStates(pcURL string) ([]ProcessState, error) {
	resp, err := http.Get(pcURL + "/processes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var states ProcessStates
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, err
	}
	return states.States, nil
}

// startProcessComposePoller polls process-compose API and publishes to NATS
func startProcessComposePoller(nc *nats.Conn) {
	pcURL := env.GetProcessComposeURL()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fmt.Printf("Starting process-compose poller (URL: %s)\n", pcURL)

	// Initial fetch
	publishProcessStates(nc, pcURL)

	for range ticker.C {
		publishProcessStates(nc, pcURL)
	}
}

// publishProcessStates fetches and publishes process states to NATS
func publishProcessStates(nc *nats.Conn, pcURL string) {
	states, err := fetchProcessStates(pcURL)
	if err != nil {
		// Silent fail - process-compose may not be running
		return
	}

	// Sort by name for stable ordering
	sort.Slice(states, func(i, j int) bool {
		return states[i].Name < states[j].Name
	})

	body, err := json.Marshal(states)
	if err != nil {
		return
	}

	_ = nc.Publish(processUpdatesSubject, body)
}
