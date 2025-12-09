package main

import "fmt"

// nats_version.go - NATS backend for CSS version/variant settings
//
// CSS settings are part of UISettings stored in NATS KV.
// This file provides CSS-specific helpers that wrap the settings functions.

// getCSSVariantFromNATS retrieves the CSS variant from NATS KV
func getCSSVariantFromNATS() (string, error) {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		return "regular", err
	}
	return settings.CSSVariant, nil
}

// setCSSVariantInNATS sets the CSS variant in NATS KV
func setCSSVariantInNATS(variant string) error {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		settings = UISettings{
			ViewportMode: "responsive",
			RTLEnabled:   false,
			RTLLang:      "ar",
		}
	}
	settings.CSSVariant = variant
	return setUISettingsInNATS(settings)
}

// getViewportModeFromNATS retrieves the viewport mode from NATS KV
func getViewportModeFromNATS() (string, error) {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		return "responsive", err
	}
	return settings.ViewportMode, nil
}

// setViewportModeInNATS sets the viewport mode in NATS KV
func setViewportModeInNATS(mode string) error {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		settings = UISettings{
			CSSVariant: "regular",
			RTLEnabled: false,
			RTLLang:    "ar",
		}
	}
	settings.ViewportMode = mode
	return setUISettingsInNATS(settings)
}

// getCSSSettingsFromNATS retrieves both CSS variant and viewport mode
func getCSSSettingsFromNATS() (variant, viewport string, err error) {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		return "regular", "responsive", err
	}
	return settings.CSSVariant, settings.ViewportMode, nil
}

// setCSSSettingsInNATS sets both CSS variant and viewport mode
func setCSSSettingsInNATS(variant, viewport string) error {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		settings = UISettings{
			RTLEnabled: false,
			RTLLang:    "ar",
		}
	}
	settings.CSSVariant = variant
	settings.ViewportMode = viewport
	return setUISettingsInNATS(settings)
}

// getLiveCSSSettings returns cached CSS settings (thread-safe)
func getLiveCSSSettings() (variant, viewport string) {
	settingsMu.RLock()
	defer settingsMu.RUnlock()
	return liveUISettings.CSSVariant, liveUISettings.ViewportMode
}

// getCDNLinkForSettings returns the Pico CSS CDN link for given settings
func getCDNLinkForSettings(variant, viewport string) string {
	v := "pico"
	if variant == "classless" {
		v = "pico.classless"
	}
	vp := ""
	if viewport == "fluid" {
		vp = ".fluid"
	}
	return fmt.Sprintf("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/%s%s.min.css", v, vp)
}
