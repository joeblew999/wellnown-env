// nats-node: NATS JetStream hub/leaf node using wellknown-env SDK
//
// This is the NATS infrastructure binary for the platform.
// Run it as:
//   - Hub (standalone): NATS_PORT=4222 NATS_NAME=hub ./nats-node
//   - Leaf node: NATS_HUB=nats://hub:4222 NATS_PORT=4223 ./nats-node
//
// The SDK (pkg/env) handles:
//   - Embedded NATS JetStream server
//   - Auth lifecycle (none/token/nkey/jwt)
//   - Service registration + heartbeat
//   - KV bucket management
//
// This binary adds:
//   - Process-compose polling and publishing
//   - Service listing on startup
//   - Logging for hub operations
//
// Environment:
//   NATS_NAME  - Node name (default: random)
//   NATS_PORT  - Client port (default: random)
//   NATS_HUB   - Hub URL for leaf mode (empty = standalone)
//   NATS_DATA  - Data directory (empty = in-memory)
//   NATS_AUTH  - Auth mode: none, token, nkey, jwt
//   PC_URL     - Process-compose API URL (default: http://localhost:8181)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/joeblew999/wellnown-env/pkg/env"
	"github.com/joeblew999/wellnown-env/pkg/env/registry"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config for nats-node specific settings
type Config struct {
	PCInterval int `conf:"default:2,env:PC_POLL_INTERVAL"` // Process-compose poll interval in seconds
}

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
	// Create manager - this starts embedded NATS automatically
	// We disable the GUI since this is infrastructure, not a service
	mgr, err := env.New("NATS_NODE", env.WithoutGUI())
	if err != nil {
		return fmt.Errorf("creating manager: %w", err)
	}
	defer mgr.Close()

	// Parse config (this also resolves secrets and registers to mesh)
	var cfg Config
	if help, err := mgr.Parse(&cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	} else if help != "" {
		fmt.Println(help)
		return nil
	}

	// Get NATS components from manager
	nc := mgr.NC()
	kv := mgr.KV()

	fmt.Printf("\nNATS node ready!\n")
	fmt.Printf("  Client URL: %s\n", mgr.ClientURL())
	if reg := mgr.Registration(); reg != nil {
		fmt.Printf("  Instance:   %s\n", reg.Instance.ID)
	}
	fmt.Println()

	// Watch for all service registrations
	watcher, err := env.WatchAll(kv, func(key string, reg *registry.ServiceRegistration, deleted bool) {
		op := "PUT"
		if deleted {
			op = "DEL"
		}
		if reg != nil {
			fmt.Printf("[WATCH] %s %s (%s/%s)\n", op, key, reg.GitHub.Org, reg.GitHub.Repo)
		} else {
			fmt.Printf("[WATCH] %s %s\n", op, key)
		}
	})
	if err != nil {
		return fmt.Errorf("watching services: %w", err)
	}
	defer watcher.Stop()

	// Start process-compose poller
	go startProcessComposePoller(nc, time.Duration(cfg.PCInterval)*time.Second)

	// Periodically list all registered services
	go listServicesLoop(kv)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	return nil
}

// listServicesLoop periodically lists all registered services
func listServicesLoop(kv jetstream.KeyValue) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		keys, err := kv.Keys(ctx)
		cancel()
		if err != nil {
			continue
		}

		fmt.Println("\n--- Registered Services ---")
		for _, k := range keys {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			entry, err := kv.Get(ctx, k)
			cancel()
			if err != nil {
				continue
			}
			fmt.Printf("  %s: %s\n", k, string(entry.Value()))
		}
		fmt.Println("---------------------------\n")
	}
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
func startProcessComposePoller(nc *nats.Conn, interval time.Duration) {
	pcURL := env.GetProcessComposeURL()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("Starting process-compose poller (URL: %s, interval: %v)\n", pcURL, interval)

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
