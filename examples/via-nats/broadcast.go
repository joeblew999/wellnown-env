package main

import (
	"sync"
)

// Topic names for broadcast channels
const (
	TopicAuth      = "auth"
	TopicMesh      = "mesh"
	TopicTests     = "tests"
	TopicServices  = "services"
	TopicNats      = "nats"
	TopicProcesses = "processes"
)

// BroadcastHub manages sync function registrations per topic
type BroadcastHub struct {
	mu       sync.RWMutex
	channels map[string]map[int]func()
	nextID   int
}

var broadcast = &BroadcastHub{
	channels: make(map[string]map[int]func()),
}

// Subscribe registers a sync function for a topic, returns unsubscribe function
func (b *BroadcastHub) Subscribe(topic string, syncFn func()) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.channels[topic] == nil {
		b.channels[topic] = make(map[int]func())
	}

	id := b.nextID
	b.nextID++
	b.channels[topic][id] = syncFn

	// Return unsubscribe function
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.channels[topic], id)
	}
}

// Notify calls all sync functions for a topic
func (b *BroadcastHub) Notify(topic string) {
	b.mu.RLock()
	fns := make([]func(), 0, len(b.channels[topic]))
	for _, fn := range b.channels[topic] {
		fns = append(fns, fn)
	}
	b.mu.RUnlock()

	// Call sync functions outside of lock
	for _, fn := range fns {
		fn()
	}
}
