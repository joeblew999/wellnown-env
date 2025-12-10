package main

import (
	"context"
	"sync"
	"time"
)

// Background state for processes
var (
	processesMu    sync.RWMutex
	liveProcesses  []ProcessState
	processesError string
)

// Background state for time
var (
	timeMu      sync.RWMutex
	currentTime string
)

// startBackgroundTickers starts background goroutines for non-NATS polling
// These emit to broadcast channels so pages don't need individual timers
func startBackgroundTickers(ctx context.Context) {
	// Time ticker - every second
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				timeMu.Lock()
				currentTime = time.Now().Format("15:04:05")
				timeMu.Unlock()
				broadcast.Notify(TopicTime)
			}
		}
	}()

	// Note: Process polling moved to nats-node (hub)
	// Via receives process updates via NATS subscription (startProcessUpdatesSubscription)
}
