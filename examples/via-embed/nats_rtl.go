package main

// nats_rtl.go - NATS backend for RTL (Right-to-Left) settings
//
// RTL settings are part of UISettings stored in NATS KV.
// This file provides RTL-specific helpers that wrap the settings functions.

// getRTLFromNATS retrieves RTL settings from NATS KV
func getRTLFromNATS() (enabled bool, lang string, err error) {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		return false, "ar", err
	}
	return settings.RTLEnabled, settings.RTLLang, nil
}

// setRTLInNATS saves RTL settings to NATS KV
func setRTLInNATS(enabled bool, lang string) error {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		// Use defaults on error
		settings = UISettings{
			CSSVariant:   "regular",
			ViewportMode: "responsive",
		}
	}
	settings.RTLEnabled = enabled
	settings.RTLLang = lang
	return setUISettingsInNATS(settings)
}

// toggleRTLInNATS toggles RTL mode and returns the new state
func toggleRTLInNATS() (newEnabled bool, err error) {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		settings = UISettings{
			CSSVariant:   "regular",
			ViewportMode: "responsive",
			RTLLang:      "ar",
		}
	}
	settings.RTLEnabled = !settings.RTLEnabled
	if err := setUISettingsInNATS(settings); err != nil {
		return false, err
	}
	return settings.RTLEnabled, nil
}

// setRTLLangInNATS sets the RTL language and enables RTL mode
func setRTLLangInNATS(lang string) error {
	settings, err := getUISettingsFromNATS()
	if err != nil {
		settings = UISettings{
			CSSVariant:   "regular",
			ViewportMode: "responsive",
		}
	}
	settings.RTLLang = lang
	settings.RTLEnabled = true
	return setUISettingsInNATS(settings)
}

// getLiveRTL returns the current cached RTL settings (thread-safe)
func getLiveRTL() (enabled bool, lang string) {
	settingsMu.RLock()
	defer settingsMu.RUnlock()
	return liveUISettings.RTLEnabled, liveUISettings.RTLLang
}
