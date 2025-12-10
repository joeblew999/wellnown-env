// discovery.go: Service discovery via NATS KV watches
//
// Enables services to:
// - Watch for changes to specific services (by org/repo)
// - Get current instances of a service
// - List all registered services
//
// Uses NATS KV watch for push-based updates - no polling.
package env

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joeblew999/wellnown-env/pkg/env/registry"
	"github.com/nats-io/nats.go/jetstream"
)

// Watcher represents an active watch on service registrations
type Watcher interface {
	// Stop stops the watcher
	Stop() error
}

// ServiceWatcher watches for changes to services
type ServiceWatcher struct {
	kvWatcher jetstream.KeyWatcher
	stopCh    chan struct{}
}

// Stop stops the watcher
func (w *ServiceWatcher) Stop() error {
	close(w.stopCh)
	return w.kvWatcher.Stop()
}

// WatchService watches for changes to a specific service (org/repo)
// The callback is called whenever any instance of the service changes
func WatchService(kv jetstream.KeyValue, name string, fn func(registry.ServiceRegistration)) (*ServiceWatcher, error) {
	// Convert org/repo to key pattern: org.repo.*
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid service name %q, expected org/repo", name)
	}
	pattern := parts[0] + "." + parts[1] + ".*"

	ctx := context.Background()
	watcher, err := kv.Watch(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("watching service %s: %w", name, err)
	}

	sw := &ServiceWatcher{
		kvWatcher: watcher,
		stopCh:    make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-sw.stopCh:
				return
			case entry := <-watcher.Updates():
				if entry == nil {
					continue
				}
				// Skip deletes for the callback
				if entry.Operation() == jetstream.KeyValueDelete {
					continue
				}

				var reg registry.ServiceRegistration
				if err := json.Unmarshal(entry.Value(), &reg); err != nil {
					continue
				}
				fn(reg)
			}
		}
	}()

	return sw, nil
}

// WatchAll watches for all service registration changes
func WatchAll(kv jetstream.KeyValue, fn func(key string, reg *registry.ServiceRegistration, deleted bool)) (*ServiceWatcher, error) {
	ctx := context.Background()
	watcher, err := kv.WatchAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("watching all services: %w", err)
	}

	sw := &ServiceWatcher{
		kvWatcher: watcher,
		stopCh:    make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-sw.stopCh:
				return
			case entry := <-watcher.Updates():
				if entry == nil {
					continue
				}

				deleted := entry.Operation() == jetstream.KeyValueDelete
				if deleted {
					fn(entry.Key(), nil, true)
					continue
				}

				var reg registry.ServiceRegistration
				if err := json.Unmarshal(entry.Value(), &reg); err != nil {
					continue
				}
				fn(entry.Key(), &reg, false)
			}
		}
	}()

	return sw, nil
}

// GetService returns all instances of a service
func GetService(ctx context.Context, kv jetstream.KeyValue, name string) ([]registry.ServiceRegistration, error) {
	// Convert org/repo to key pattern
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid service name %q, expected org/repo", name)
	}
	prefix := parts[0] + "." + parts[1] + "."

	keys, err := kv.Keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing keys: %w", err)
	}

	var registrations []registry.ServiceRegistration
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		entry, err := kv.Get(ctx, key)
		if err != nil {
			continue
		}

		var reg registry.ServiceRegistration
		if err := json.Unmarshal(entry.Value(), &reg); err != nil {
			continue
		}
		registrations = append(registrations, reg)
	}

	return registrations, nil
}

// GetAllServices returns all registered services
func GetAllServices(ctx context.Context, kv jetstream.KeyValue) ([]registry.ServiceRegistration, error) {
	keys, err := kv.Keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing keys: %w", err)
	}

	var registrations []registry.ServiceRegistration
	for _, key := range keys {
		entry, err := kv.Get(ctx, key)
		if err != nil {
			continue
		}

		var reg registry.ServiceRegistration
		if err := json.Unmarshal(entry.Value(), &reg); err != nil {
			continue
		}
		registrations = append(registrations, reg)
	}

	return registrations, nil
}

// ServiceExists checks if at least one instance of a service exists
func ServiceExists(ctx context.Context, kv jetstream.KeyValue, name string) (bool, error) {
	instances, err := GetService(ctx, kv, name)
	if err != nil {
		return false, err
	}
	return len(instances) > 0, nil
}
