package main

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// startMonitorSubscription subscribes to NATS subjects for monitoring
func startMonitorSubscription(pattern string) error {
	nc, err := getNatsConn()
	if err != nil {
		return err
	}

	// Stop existing subscription if any
	stopMonitorSubscription()

	// Initialize stats
	monitorMu.Lock()
	monitorStats = MonitorStats{
		SubjectsSeen: make(map[string]int64),
		StartTime:    time.Now(),
	}
	monitorMessages = nil
	monitorMu.Unlock()

	sub, err := nc.Subscribe(pattern, func(msg *nats.Msg) {
		monitorMu.Lock()
		defer monitorMu.Unlock()

		// Truncate data for display if too long
		data := string(msg.Data)
		if len(data) > 200 {
			data = data[:200] + "..."
		}

		// Create monitor message
		monMsg := MonitorMessage{
			Subject: msg.Subject,
			Data:    data,
			Size:    len(msg.Data),
			Time:    time.Now(),
		}

		// Add to messages (keep last 100)
		monitorMessages = append(monitorMessages, monMsg)
		if len(monitorMessages) > 100 {
			monitorMessages = monitorMessages[len(monitorMessages)-100:]
		}

		// Update stats
		monitorStats.TotalMessages++
		monitorStats.LastMessage = time.Now()
		monitorStats.SubjectsSeen[msg.Subject]++

		// Notify all subscribed clients (outside of lock via defer)
		defer broadcast.Notify(TopicMonitor)
	})
	if err != nil {
		return fmt.Errorf("subscribing to %s: %w", pattern, err)
	}

	monitorMu.Lock()
	monitorSub = sub
	monitorPattern = pattern
	monitorMu.Unlock()

	fmt.Printf("[MONITOR] Subscribed to pattern: %s\n", pattern)
	return nil
}

// stopMonitorSubscription unsubscribes from the monitor subscription
func stopMonitorSubscription() {
	monitorMu.Lock()
	defer monitorMu.Unlock()

	if monitorSub != nil {
		_ = monitorSub.Unsubscribe()
		monitorSub = nil
		fmt.Println("[MONITOR] Unsubscribed")
	}
}

// getMonitorMessages returns a copy of the current monitor messages
func getMonitorMessages() []MonitorMessage {
	monitorMu.RLock()
	defer monitorMu.RUnlock()

	msgs := make([]MonitorMessage, len(monitorMessages))
	copy(msgs, monitorMessages)
	return msgs
}

// getMonitorStats returns a copy of the current monitor stats
func getMonitorStats() MonitorStats {
	monitorMu.RLock()
	defer monitorMu.RUnlock()

	stats := monitorStats
	// Deep copy the map
	stats.SubjectsSeen = make(map[string]int64)
	for k, v := range monitorStats.SubjectsSeen {
		stats.SubjectsSeen[k] = v
	}
	return stats
}

// clearMonitorMessages clears all captured messages
func clearMonitorMessages() {
	monitorMu.Lock()
	defer monitorMu.Unlock()

	monitorMessages = nil
	monitorStats.TotalMessages = 0
	monitorStats.SubjectsSeen = make(map[string]int64)
	monitorStats.StartTime = time.Now()
}

// isMonitorActive returns whether the monitor subscription is active
func isMonitorActive() bool {
	monitorMu.RLock()
	defer monitorMu.RUnlock()
	return monitorSub != nil
}

// getMonitorPattern returns the current subscription pattern
func getMonitorPattern() string {
	monitorMu.RLock()
	defer monitorMu.RUnlock()
	return monitorPattern
}
