package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Live service registry state
var (
	servicesMu   sync.RWMutex
	liveServices []ServiceRegistration
)

// getServicesFromNATS fetches all registered services from NATS KV
func getServicesFromNATS() ([]ServiceRegistration, error) {
	js, err := getNatsJS()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Get services_registry KV bucket
	kv, err := js.KeyValue(ctx, "services_registry")
	if err != nil {
		return nil, fmt.Errorf("getting services_registry: %w", err)
	}

	keys, err := kv.Keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing keys: %w", err)
	}

	var services []ServiceRegistration
	for _, key := range keys {
		entry, err := kv.Get(ctx, key)
		if err != nil {
			continue
		}
		var svc ServiceRegistration
		if err := json.Unmarshal(entry.Value(), &svc); err != nil {
			continue
		}
		services = append(services, svc)
	}

	return services, nil
}

// registerViaService registers this via instance in the services_registry KV bucket
func registerViaService(ctx context.Context) error {
	js, err := getNatsJS()
	if err != nil {
		return err
	}

	// Get or create services_registry KV bucket
	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      "services_registry",
		Description: "Service registration for wellnown-env",
		TTL:         30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("getting services_registry: %w", err)
	}

	// Register this via instance
	viaPort := getViaPort()
	key := fmt.Sprintf("via.web.%s", viaPort)
	registration := ServiceRegistration{
		Name: "via-web",
		Host: fmt.Sprintf("http://localhost:%s", viaPort),
		Time: time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(registration)
	if err != nil {
		return fmt.Errorf("marshaling registration: %w", err)
	}

	_, err = kv.Put(ctx, key, data)
	if err != nil {
		return fmt.Errorf("registering service: %w", err)
	}

	fmt.Printf("Registered via-web service: %s\n", key)

	// Start heartbeat to keep registration alive
	go startServiceHeartbeat(ctx, kv, key, registration)

	return nil
}

// startServiceHeartbeat keeps the service registration alive by updating it periodically
func startServiceHeartbeat(ctx context.Context, kv jetstream.KeyValue, key string, reg ServiceRegistration) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Deregister on shutdown
			if err := kv.Delete(ctx, key); err != nil {
				fmt.Printf("Failed to deregister %s: %v\n", key, err)
			} else {
				fmt.Printf("Deregistered: %s\n", key)
			}
			return
		case <-ticker.C:
			reg.Time = time.Now().Format(time.RFC3339)
			data, err := json.Marshal(reg)
			if err != nil {
				continue
			}
			if _, err := kv.Put(ctx, key, data); err != nil {
				fmt.Printf("Heartbeat failed for %s: %v\n", key, err)
			}
		}
	}
}

// watchServicesChanges watches for changes in services_registry KV
func watchServicesChanges(ctx context.Context) {
	js, err := getNatsJS()
	if err != nil {
		return
	}

	// Get services_registry KV bucket
	kv, err := js.KeyValue(ctx, "services_registry")
	if err != nil {
		fmt.Printf("Failed to get services_registry for watching: %v\n", err)
		return
	}

	// Watch all keys in the bucket
	watcher, err := kv.WatchAll(ctx)
	if err != nil {
		fmt.Printf("Failed to watch services_registry: %v\n", err)
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-watcher.Updates():
				if entry == nil {
					continue
				}

				// Refresh the full list
				services, err := getServicesFromNATS()
				if err == nil {
					servicesMu.Lock()
					liveServices = services
					servicesMu.Unlock()
				}

				fmt.Printf("Services registry updated: %d services\n", len(services))
				broadcast.Notify(TopicNats)
			}
		}
	}()
}
