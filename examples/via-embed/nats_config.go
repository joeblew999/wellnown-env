package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
)

// Config keys and their toggle values
var configToggle = map[string][]string{
	"app.name":            {"MyApp", "MyApp-v2", "MyApp-prod"},
	"app.debug":           {"true", "false"},
	"app.log_level":       {"debug", "info", "warn", "error"},
	"feature.flag1":       {"enabled", "disabled"},
	"feature.flag2":       {"enabled", "disabled"},
	"service.timeout":     {"5s", "10s", "30s", "60s"},
	"service.retry_count": {"3", "5", "10"},
}

// getConfig fetches a config value from NATS KV
func getConfig(key string) string {
	kv, err := getNatsKV()
	if err != nil {
		return ""
	}
	entry, err := kv.Get(context.Background(), "config."+key)
	if err != nil {
		return ""
	}
	return string(entry.Value())
}

// setConfig sets a config value in NATS KV
func setConfig(key, value string) {
	kv, err := getNatsKV()
	if err != nil {
		return
	}
	kv.Put(context.Background(), "config."+key, []byte(value))
}

// toggleConfig cycles through possible values for a key
func toggleConfig(key string) {
	current := getConfig(key)
	values := configToggle[key]
	if len(values) == 0 {
		return
	}
	// Find next value
	next := values[0]
	for i, v := range values {
		if v == current && i+1 < len(values) {
			next = values[i+1]
			break
		}
	}
	setConfig(key, next)
}

// deleteConfig removes a config key from NATS KV
func deleteConfig(key string) {
	kv, err := getNatsKV()
	if err != nil {
		return
	}
	kv.Delete(context.Background(), "config."+key)
}

// watchConfigChanges watches NATS KV for config changes and notifies subscribers
func watchConfigChanges(ctx context.Context) {
	kv, err := getNatsKV()
	if err != nil {
		return
	}
	watcher, err := kv.Watch(ctx, "config.>")
	if err != nil {
		fmt.Printf("Error watching config: %v\n", err)
		return
	}
	for entry := range watcher.Updates() {
		if entry == nil {
			continue
		}
		key := strings.TrimPrefix(entry.Key(), "config.")
		if entry.Operation() == jetstream.KeyValueDelete {
			fmt.Printf("[CONFIG] %s deleted\n", key)
		} else {
			fmt.Printf("[CONFIG] %s = %s\n", key, string(entry.Value()))
		}
		broadcast.Notify(TopicConfig)
	}
}
