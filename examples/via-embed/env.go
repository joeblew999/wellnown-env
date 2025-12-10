package main

import (
	"fmt"

	"github.com/go-via/via-plugin-picocss/picocss"
	"github.com/joeblew999/wellnown-env/pkg/env"
	"github.com/joeblew999/wellnown-env/pkg/viatheme"
)

// =============================================================================
// Configuration
// =============================================================================
// All defaults match .env file (single source of truth).
// Environment variables override defaults at runtime.

// Default values - MUST match .env file
const (
	defaultNatsURL      = "nats://localhost:4222"
	defaultNatsKVConfig = "via_config"
	defaultViaPort      = "3000"
	defaultViaTheme     = "purple"
	defaultPCAddress    = "localhost"
	defaultPCPort       = "8181"
)

// ThemeNames is exported for use in UI components
var ThemeNames = viatheme.ThemeNames

// getThemeFromEnv reads VIA_THEME env var and returns the corresponding theme
func getThemeFromEnv() (picocss.Theme, string) {
	return viatheme.GetFromEnv(defaultViaTheme)
}

// getViaPort returns the Via server port (VIA_PORT env or default)
func getViaPort() string {
	return env.GetEnv("VIA_PORT", defaultViaPort)
}

// getViaAddress returns the Via server address in :port format
func getViaAddress() string {
	return ":" + getViaPort()
}

// getProcessComposeURL constructs URL from PC_ADDRESS and PC_PORT env vars
func getProcessComposeURL() string {
	return env.GetProcessComposeURL()
}

// getNatsURL returns NATS server URL (NATS_URL env or default)
func getNatsURL() string {
	return env.GetEnv("NATS_URL", defaultNatsURL)
}

// getNatsKVBucket returns the NATS KV bucket name for config (NATS_KV_CONFIG env or default)
func getNatsKVBucket() string {
	return env.GetEnv("NATS_KV_CONFIG", defaultNatsKVConfig)
}

// formatPCAddress formats the PC_ADDRESS and PC_PORT for display
func formatPCAddress() string {
	addr := env.GetEnv("PC_ADDRESS", defaultPCAddress)
	port := env.GetEnv("PC_PORT", defaultPCPort)
	return fmt.Sprintf("%s:%s", addr, port)
}
