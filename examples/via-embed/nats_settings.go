package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// getUISettingsFromNATS retrieves UI settings from NATS KV
func getUISettingsFromNATS() (UISettings, error) {
	settings := UISettings{
		CSSVariant:   "regular",    // default
		ViewportMode: "responsive", // default (container-based)
		RTLEnabled:   false,
		RTLLang:      "ar",
	}

	kv, err := getNatsKV()
	if err != nil {
		return settings, err
	}

	// Try to get settings from KV
	entry, err := kv.Get(context.Background(), "ui_settings")
	if err != nil {
		// Not found is ok, return defaults
		return settings, nil
	}

	if err := json.Unmarshal(entry.Value(), &settings); err != nil {
		return settings, fmt.Errorf("failed to parse settings: %w", err)
	}

	return settings, nil
}

// setUISettingsInNATS saves UI settings to NATS KV
func setUISettingsInNATS(settings UISettings) error {
	kv, err := getNatsKV()
	if err != nil {
		return err
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	_, err = kv.Put(context.Background(), "ui_settings", data)
	if err != nil {
		return fmt.Errorf("failed to put settings: %w", err)
	}

	// Update local state
	settingsMu.Lock()
	liveUISettings = settings
	settingsMu.Unlock()

	return nil
}

// watchUISettingsChanges watches for UI settings changes from NATS KV
func watchUISettingsChanges(ctx context.Context) {
	kv, err := getNatsKV()
	if err != nil {
		return
	}

	watcher, err := kv.Watch(ctx, "ui_settings")
	if err != nil {
		fmt.Printf("Failed to watch ui_settings: %v\n", err)
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-watcher.Updates():
				if entry == nil {
					continue
				}

				var settings UISettings
				if err := json.Unmarshal(entry.Value(), &settings); err != nil {
					continue
				}

				settingsMu.Lock()
				liveUISettings = settings
				settingsMu.Unlock()

				fmt.Printf("UI settings updated via NATS: variant=%s, viewport=%s, rtl=%v\n",
					settings.CSSVariant, settings.ViewportMode, settings.RTLEnabled)
				// Notify all subscribed clients
				broadcast.Notify(TopicSettings)
			}
		}
	}()
}
