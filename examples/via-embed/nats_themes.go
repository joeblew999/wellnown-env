package main

import (
	"context"
	"fmt"
)

// Available themes for the UI
var themes = []string{
	"amber", "blue", "cyan", "fuchsia", "green",
	"indigo", "jade", "lime", "orange", "pink",
	"pumpkin", "purple", "red", "sand", "slate",
	"violet", "yellow", "zinc",
}

// getTheme fetches the current theme from NATS KV
func getTheme() string {
	kv, err := getNatsKV()
	if err != nil {
		return ""
	}
	entry, err := kv.Get(context.Background(), "theme")
	if err != nil {
		return ""
	}
	return string(entry.Value())
}

// setTheme updates the theme in NATS KV
func setTheme(name string) {
	kv, err := getNatsKV()
	if err != nil {
		return
	}
	kv.Put(context.Background(), "theme", []byte(name))
}

// watchThemeChanges watches NATS KV for theme changes and notifies subscribers
func watchThemeChanges(ctx context.Context) {
	kv, err := getNatsKV()
	if err != nil {
		return
	}
	watcher, err := kv.Watch(ctx, "theme")
	if err != nil {
		fmt.Printf("Error watching theme: %v\n", err)
		return
	}
	for entry := range watcher.Updates() {
		if entry == nil {
			continue
		}
		fmt.Printf("[THEME] %s\n", string(entry.Value()))
		broadcast.Notify(TopicTheme)
	}
}
