package main

import (
	"errors"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ErrNatsNotConnected is returned when NATS operations are attempted without a connection
var ErrNatsNotConnected = errors.New("not connected to NATS")

// Global NATS state (shared across all Via pages)
var (
	natsConn      *nats.Conn
	natsJS        jetstream.JetStream
	natsKV        jetstream.KeyValue
	natsConnected bool
	natsMu        sync.RWMutex

	// Chat messages via NATS pub/sub
	chatMessages []ChatMessage
	chatMu       sync.RWMutex
	chatSub      *nats.Subscription

	// NATS message monitor state
	monitorMessages []MonitorMessage
	monitorMu       sync.RWMutex
	monitorSub      *nats.Subscription
	monitorStats    MonitorStats
	monitorPattern  string

	// Process state via NATS (pc.processes updates)
	processesNATSMu     sync.RWMutex
	processesNATS       []ProcessState
	processesNATSError  string
	processesUpdatesSub *nats.Subscription
	processesControlMu  sync.RWMutex

	// UI Settings from NATS KV (version picker, RTL)
	liveUISettings UISettings
	settingsMu     sync.RWMutex
)

// getNatsKV returns the NATS KV store and an error if not connected
func getNatsKV() (jetstream.KeyValue, error) {
	natsMu.RLock()
	kv, connected := natsKV, natsConnected
	natsMu.RUnlock()
	if !connected || kv == nil {
		return nil, ErrNatsNotConnected
	}
	return kv, nil
}

// getNatsConn returns the NATS connection and an error if not connected
func getNatsConn() (*nats.Conn, error) {
	natsMu.RLock()
	nc, connected := natsConn, natsConnected
	natsMu.RUnlock()
	if !connected || nc == nil {
		return nil, ErrNatsNotConnected
	}
	return nc, nil
}

// getNatsJS returns the NATS JetStream context and an error if not connected
func getNatsJS() (jetstream.JetStream, error) {
	natsMu.RLock()
	js, connected := natsJS, natsConnected
	natsMu.RUnlock()
	if !connected || js == nil {
		return nil, ErrNatsNotConnected
	}
	return js, nil
}

// isNatsConnected returns the current NATS connection status
func isNatsConnected() bool {
	natsMu.RLock()
	connected := natsConnected
	natsMu.RUnlock()
	return connected
}
