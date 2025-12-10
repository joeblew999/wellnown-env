// nats.go: Embedded NATS JetStream server and client connection
//
// Every service using wellknown-env embeds a NATS leaf node that:
// - Works completely offline when hub is unavailable
// - Automatically syncs with hub when connectivity is restored
// - Persists data locally (if DataDir is set)
// - Supports the full auth lifecycle (none → token → nkey → jwt)
//
// Topology:
//
//	STANDALONE: Service runs its own NATS (dev mode, no hub)
//	LEAF:       Service connects to a hub cluster (production)
//
// The embedded NATS provides:
// - JetStream for persistence and KV
// - Service registry via KV bucket
// - Event streaming via subjects
package env

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATSConfig holds NATS server configuration
type NATSConfig struct {
	Name    string // Server name
	Port    int    // Client port (0 = random)
	HubURL  string // Hub URL for leaf mode (empty = standalone)
	DataDir string // Data directory (empty = in-memory)
}

// NATSNode wraps an embedded NATS server and client connection
type NATSNode struct {
	server *server.Server
	conn   *nats.Conn
	js     jetstream.JetStream
	kv     jetstream.KeyValue
	config NATSConfig
}

// StartNATSNode creates and starts an embedded NATS server
func StartNATSNode(cfg NATSConfig, authCfg *AuthConfig) (*NATSNode, error) {
	// Default name if not provided
	if cfg.Name == "" {
		cfg.Name = "node-" + uuid.New().String()[:8]
	}

	// Configure server options
	opts := &server.Options{
		ServerName: cfg.Name,
		Port:       cfg.Port,
		JetStream:  true,
		StoreDir:   cfg.DataDir,
		NoLog:      true, // Quiet by default, apps can enable logging
		Debug:      false,
		Trace:      false,
	}

	// Configure authentication if provided
	if authCfg != nil {
		if err := ConfigureAuth(opts, authCfg); err != nil {
			return nil, fmt.Errorf("configuring auth: %w", err)
		}
	}

	// Configure as leaf node if hub URL provided
	if cfg.HubURL != "" {
		u, err := url.Parse(cfg.HubURL)
		if err != nil {
			return nil, fmt.Errorf("parsing hub URL: %w", err)
		}
		opts.LeafNode = server.LeafNodeOpts{
			Remotes: []*server.RemoteLeafOpts{
				{URLs: []*url.URL{u}},
			},
		}
	} else {
		// Enable leaf node listening so other nodes can connect
		if cfg.Port > 0 {
			opts.LeafNode = server.LeafNodeOpts{
				Port: cfg.Port + 1000, // Leaf port = client port + 1000
			}
		}
	}

	// Create and start the embedded server
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("creating server: %w", err)
	}

	// Start in background
	go ns.Start()

	// Wait for server to be ready
	if !ns.ReadyForConnections(15 * time.Second) {
		return nil, fmt.Errorf("server not ready within 15s")
	}

	// Connect as a client to our own embedded server
	var connOpts []nats.Option
	if authCfg != nil {
		clientOpts, err := GetClientConnectOptions(authCfg)
		if err != nil {
			ns.Shutdown()
			return nil, fmt.Errorf("getting client auth options: %w", err)
		}
		connOpts = clientOpts
	}

	nc, err := nats.Connect(ns.ClientURL(), connOpts...)
	if err != nil {
		ns.Shutdown()
		return nil, fmt.Errorf("connecting to server: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		ns.Shutdown()
		return nil, fmt.Errorf("creating jetstream: %w", err)
	}

	// Create the services_registry KV bucket
	ctx := context.Background()
	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      "services_registry",
		Description: "Service registration for wellnown-env",
		TTL:         30 * time.Second, // Entries expire if not refreshed
	})
	if err != nil {
		nc.Close()
		ns.Shutdown()
		return nil, fmt.Errorf("creating KV bucket: %w", err)
	}

	return &NATSNode{
		server: ns,
		conn:   nc,
		js:     js,
		kv:     kv,
		config: cfg,
	}, nil
}

// ClientURL returns the NATS client URL
func (n *NATSNode) ClientURL() string {
	return n.server.ClientURL()
}

// Conn returns the NATS connection
func (n *NATSNode) Conn() *nats.Conn {
	return n.conn
}

// JetStream returns the JetStream context
func (n *NATSNode) JetStream() jetstream.JetStream {
	return n.js
}

// KV returns the services_registry KV bucket
func (n *NATSNode) KV() jetstream.KeyValue {
	return n.kv
}

// Name returns the server name
func (n *NATSNode) Name() string {
	return n.config.Name
}

// IsLeaf returns true if connected to a hub
func (n *NATSNode) IsLeaf() bool {
	return n.config.HubURL != ""
}

// Close shuts down the NATS node gracefully
func (n *NATSNode) Close() error {
	if n.conn != nil {
		n.conn.Close()
	}
	if n.server != nil {
		n.server.Shutdown()
		n.server.WaitForShutdown()
	}
	return nil
}

// Publish publishes a message to a subject
func (n *NATSNode) Publish(subject string, data []byte) error {
	return n.conn.Publish(subject, data)
}

// Subscribe subscribes to a subject
func (n *NATSNode) Subscribe(subject string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return n.conn.Subscribe(subject, handler)
}

// QueueSubscribe subscribes to a subject with a queue group
func (n *NATSNode) QueueSubscribe(subject, queue string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return n.conn.QueueSubscribe(subject, queue, handler)
}
