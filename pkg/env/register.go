// register.go: Service registration to NATS KV with heartbeat
//
// When Parse() is called, the service:
// 1. Extracts config fields via reflection
// 2. Builds a ServiceRegistration with GitHub identity + fields
// 3. Stores it in NATS KV bucket "services_registry"
// 4. Starts a heartbeat goroutine to keep registration alive
//
// Key format: {org}.{repo}.{instance_id}
// TTL: 30 seconds (must heartbeat every 10s)
package env

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joeblew999/wellnown-env/pkg/env/registry"
	"github.com/nats-io/nats.go/jetstream"
)

// Registrar handles service registration and heartbeat
type Registrar struct {
	mu       sync.Mutex
	kv       jetstream.KeyValue
	key      string
	reg      registry.ServiceRegistration
	stopCh   chan struct{}
	stopped  bool
	interval time.Duration
}

// NewRegistrar creates a new service registrar
func NewRegistrar(kv jetstream.KeyValue, interval time.Duration) *Registrar {
	return &Registrar{
		kv:       kv,
		stopCh:   make(chan struct{}),
		interval: interval,
	}
}

// Register creates a service registration from config struct and starts heartbeat
func (r *Registrar) Register(ctx context.Context, prefix string, cfg interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Build registration from config struct
	r.reg = registry.ServiceRegistration{
		GitHub: registry.GetGitHubInfo(),
		Instance: registry.InstanceInfo{
			ID:      uuid.New().String()[:8],
			Host:    "", // TODO: detect host:port from config
			Started: time.Now(),
		},
		Fields: ExtractFields(prefix, cfg),
	}

	// Build KV key
	if r.reg.GitHub.Org != "" && r.reg.GitHub.Repo != "" {
		r.key = r.reg.KVKey()
	} else {
		// Fallback for dev when ldflags not set
		r.key = "unknown." + r.reg.Instance.ID
	}

	// Store initial registration
	if err := r.store(ctx); err != nil {
		return err
	}

	// Start heartbeat
	go r.heartbeat()

	return nil
}

// store writes the registration to KV
func (r *Registrar) store(ctx context.Context) error {
	data, err := json.Marshal(r.reg)
	if err != nil {
		return fmt.Errorf("marshaling registration: %w", err)
	}

	_, err = r.kv.Put(ctx, r.key, data)
	if err != nil {
		return fmt.Errorf("storing registration: %w", err)
	}

	return nil
}

// heartbeat periodically refreshes the registration
func (r *Registrar) heartbeat() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.mu.Lock()
			if r.stopped {
				r.mu.Unlock()
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.store(ctx); err != nil {
				// Log but don't fail - registration will expire
				fmt.Printf("heartbeat failed: %v\n", err)
			}
			cancel()
			r.mu.Unlock()
		}
	}
}

// Deregister removes the service from the registry
func (r *Registrar) Deregister(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopped = true
	close(r.stopCh)

	if r.key != "" {
		return r.kv.Delete(ctx, r.key)
	}
	return nil
}

// Key returns the registration key
func (r *Registrar) Key() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.key
}

// Registration returns a copy of the current registration
func (r *Registrar) Registration() registry.ServiceRegistration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reg
}
