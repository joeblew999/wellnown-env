// manager.go: Core SDK manager that wraps config, secrets, NATS, and registration
//
// Usage:
//
//	type Config struct {
//	    Server struct {
//	        Port int `conf:"default:8080"`
//	    }
//	    DB struct {
//	        Password string `conf:"mask,required"`
//	    }
//	}
//
//	func main() {
//	    mgr, _ := env.New("APP")
//	    defer mgr.Close()
//
//	    var cfg Config
//	    mgr.Parse(&cfg)
//	    // Config parsed, secrets resolved, registered to mesh
//	}
package env

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/joeblew999/wellnown-env/pkg/env/registry"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Manager is the core SDK type that provides:
// - Config parsing via ardanlabs/conf
// - Secret resolution via helmfile/vals
// - NATS connectivity (embedded leaf node)
// - Service registration
type Manager struct {
	prefix string
	opts   Options

	mu        sync.RWMutex
	closed    bool
	natsNode  *NATSNode
	registrar *Registrar
}

// Options for Manager configuration
type Options struct {
	// NATS settings
	HubURL      string // NATS hub URL (empty = standalone)
	DataDir     string // Data directory (empty = in-memory)
	NATSPort    int    // NATS client port (0 = random)
	NATSName    string // Node name

	// Registration
	DisableRegistration bool // Skip service registration
	DisableHeartbeat    bool // Skip heartbeat
	HeartbeatInterval   int  // Heartbeat interval in seconds (default: 10)

	// GUI
	GUIAddr    string // GUI address (default: :3001)
	DisableGUI bool   // Disable GUI

	// Auth
	AuthMode string // none, token, nkey, jwt

	// Disable NATS completely (for simple config-only use)
	DisableNATS bool
}

// Option is a functional option for Manager
type Option func(*Options)

// WithHub sets the NATS hub URL for mesh connectivity
func WithHub(url string) Option {
	return func(o *Options) {
		o.HubURL = url
	}
}

// WithDataDir sets the NATS data directory for persistence
func WithDataDir(path string) Option {
	return func(o *Options) {
		o.DataDir = path
	}
}

// WithPort sets the NATS client port
func WithPort(port int) Option {
	return func(o *Options) {
		o.NATSPort = port
	}
}

// WithoutRegistration disables service registration
func WithoutRegistration() Option {
	return func(o *Options) {
		o.DisableRegistration = true
	}
}

// WithoutHeartbeat disables heartbeat
func WithoutHeartbeat() Option {
	return func(o *Options) {
		o.DisableHeartbeat = true
	}
}

// WithHeartbeatInterval sets custom heartbeat interval in seconds
func WithHeartbeatInterval(seconds int) Option {
	return func(o *Options) {
		o.HeartbeatInterval = seconds
	}
}

// WithGUI sets the GUI bind address
func WithGUI(addr string) Option {
	return func(o *Options) {
		o.GUIAddr = addr
	}
}

// WithoutGUI disables the GUI
func WithoutGUI() Option {
	return func(o *Options) {
		o.DisableGUI = true
	}
}

// WithoutNATS disables embedded NATS (config-only mode)
func WithoutNATS() Option {
	return func(o *Options) {
		o.DisableNATS = true
		o.DisableRegistration = true
	}
}

// New creates a new Manager with the given prefix for environment variables.
// The prefix is used by ardanlabs/conf to namespace env vars (e.g., APP_DB_PASSWORD).
func New(prefix string, opts ...Option) (*Manager, error) {
	// Build options with defaults from environment
	o := Options{
		HubURL:            os.Getenv("NATS_HUB"),
		DataDir:           os.Getenv("NATS_DATA"),
		NATSName:          GetEnv("NATS_NAME", ""),
		NATSPort:          GetEnvInt("NATS_PORT", 0),
		AuthMode:          GetEnv("NATS_AUTH", "none"),
		GUIAddr:           GetEnv("GUI_ADDR", ":3001"),
		HeartbeatInterval: GetEnvInt("HEARTBEAT_INTERVAL", 10),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(&o)
	}

	m := &Manager{
		prefix: prefix,
		opts:   o,
	}

	// Initialize embedded NATS if not disabled
	if !o.DisableNATS {
		authCfg, err := LoadAuthConfig()
		if err != nil {
			return nil, fmt.Errorf("loading auth config: %w", err)
		}

		natsCfg := NATSConfig{
			Name:    o.NATSName,
			Port:    o.NATSPort,
			HubURL:  o.HubURL,
			DataDir: o.DataDir,
		}

		node, err := StartNATSNode(natsCfg, authCfg)
		if err != nil {
			return nil, fmt.Errorf("starting NATS node: %w", err)
		}
		m.natsNode = node

		// Create registrar if registration is enabled
		if !o.DisableRegistration {
			interval := time.Duration(o.HeartbeatInterval) * time.Second
			m.registrar = NewRegistrar(node.KV(), interval)
		}
	}

	return m, nil
}

// Parse parses config from environment variables, resolves secrets,
// and registers the service to the mesh.
//
// The cfg parameter must be a pointer to a struct with conf tags.
// Returns help text if --help was provided.
func (m *Manager) Parse(cfg interface{}) (string, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return "", fmt.Errorf("manager is closed")
	}
	m.mu.RUnlock()

	// Step 1: Resolve secrets in environment BEFORE parsing config
	// This replaces ref+vault://... with actual values
	if err := ResolveEnvSecrets(); err != nil {
		return "", fmt.Errorf("resolving secrets: %w", err)
	}

	// Step 2: Parse config using ardanlabs/conf
	help, err := conf.Parse(m.prefix, cfg)
	if err != nil {
		if err == conf.ErrHelpWanted {
			return help, nil
		}
		return "", fmt.Errorf("parsing config: %w", err)
	}

	// Step 3: Register to mesh
	if m.registrar != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := m.registrar.Register(ctx, m.prefix, cfg); err != nil {
			return "", fmt.Errorf("registering service: %w", err)
		}
	}

	// Note: GUI is no longer auto-started. Services should create their own Via
	// instance and use RegisterDashboardPage/RegisterConfigPage as needed.

	return "", nil
}

// Close shuts down the manager and disconnects from NATS
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true

	// Deregister from mesh
	if m.registrar != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.registrar.Deregister(ctx); err != nil {
			// Log but don't fail - we're shutting down anyway
			fmt.Printf("deregister failed: %v\n", err)
		}
	}

	// Shutdown NATS
	if m.natsNode != nil {
		if err := m.natsNode.Close(); err != nil {
			return fmt.Errorf("closing NATS: %w", err)
		}
	}

	return nil
}

// Prefix returns the environment variable prefix
func (m *Manager) Prefix() string {
	return m.prefix
}

// GUIAddr returns the GUI address
func (m *Manager) GUIAddr() string {
	return m.opts.GUIAddr
}

// NC returns the NATS connection (nil if NATS disabled)
func (m *Manager) NC() *nats.Conn {
	if m.natsNode == nil {
		return nil
	}
	return m.natsNode.Conn()
}

// KV returns the services_registry KV bucket (nil if NATS disabled)
func (m *Manager) KV() jetstream.KeyValue {
	if m.natsNode == nil {
		return nil
	}
	return m.natsNode.KV()
}

// JetStream returns the JetStream context (nil if NATS disabled)
func (m *Manager) JetStream() jetstream.JetStream {
	if m.natsNode == nil {
		return nil
	}
	return m.natsNode.JetStream()
}

// ClientURL returns the NATS client URL (empty if NATS disabled)
func (m *Manager) ClientURL() string {
	if m.natsNode == nil {
		return ""
	}
	return m.natsNode.ClientURL()
}

// WatchService watches for changes to a specific service (org/repo)
func (m *Manager) WatchService(name string, fn func(registry.ServiceRegistration)) (Watcher, error) {
	if m.natsNode == nil {
		return nil, fmt.Errorf("NATS is disabled")
	}
	return WatchService(m.natsNode.KV(), name, fn)
}

// GetService returns all instances of a service
func (m *Manager) GetService(ctx context.Context, name string) ([]registry.ServiceRegistration, error) {
	if m.natsNode == nil {
		return nil, fmt.Errorf("NATS is disabled")
	}
	return GetService(ctx, m.natsNode.KV(), name)
}

// GetAllServices returns all registered services
func (m *Manager) GetAllServices(ctx context.Context) ([]registry.ServiceRegistration, error) {
	if m.natsNode == nil {
		return nil, fmt.Errorf("NATS is disabled")
	}
	return GetAllServices(ctx, m.natsNode.KV())
}

// OnRotate subscribes to secret rotation notifications
func (m *Manager) OnRotate(fn func(path string)) (*nats.Subscription, error) {
	if m.natsNode == nil {
		return nil, fmt.Errorf("NATS is disabled")
	}
	return OnRotate(m.natsNode.Conn(), fn)
}

// Registration returns the current service registration (nil if not registered)
func (m *Manager) Registration() *registry.ServiceRegistration {
	if m.registrar == nil {
		return nil
	}
	reg := m.registrar.Registration()
	return &reg
}
