package main

import (
	"context"
	"fmt"
	"strconv"
)

// getCounter fetches the current counter value from NATS KV
func getCounter() int64 {
	kv, err := getNatsKV()
	if err != nil {
		return 0
	}
	entry, err := kv.Get(context.Background(), "counter")
	if err != nil {
		return 0
	}
	val, _ := strconv.ParseInt(string(entry.Value()), 10, 64)
	return val
}

// setCounter updates the counter value in NATS KV
func setCounter(value int64) {
	kv, err := getNatsKV()
	if err != nil {
		return
	}
	kv.Put(context.Background(), "counter", []byte(strconv.FormatInt(value, 10)))
	// Note: watchCounterChanges will notify via broadcast when KV updates
}

// watchCounterChanges watches NATS KV for counter changes and notifies subscribers
func watchCounterChanges(ctx context.Context) {
	kv, err := getNatsKV()
	if err != nil {
		return
	}
	watcher, err := kv.Watch(ctx, "counter")
	if err != nil {
		fmt.Printf("Error watching counter: %v\n", err)
		return
	}
	for entry := range watcher.Updates() {
		if entry == nil {
			continue
		}
		broadcast.Notify(TopicCounter)
	}
}
