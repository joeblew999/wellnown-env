package main

import (
	"github.com/go-via/via-plugin-picocss/picocss"
	"github.com/joeblew999/wellnown-env/pkg/env"
	"github.com/joeblew999/wellnown-env/pkg/viatheme"
)

// Default values
const (
	defaultNatsURL     = "nats://localhost:4222"
	defaultViaPort     = "3001"
	defaultViaTheme    = "indigo"
	defaultNatsNodeDir = "../nats-node"
)

// getThemeFromEnv reads VIA_THEME env var and returns the corresponding theme
func getThemeFromEnv() (picocss.Theme, string) {
	return viatheme.GetFromEnv(defaultViaTheme)
}

// getViaPort returns the Via server port
func getViaPort() string {
	return env.GetEnv("VIA_PORT", defaultViaPort)
}

// getViaAddress returns the Via server address in :port format
func getViaAddress() string {
	return ":" + getViaPort()
}

// getNatsURL returns NATS server URL
func getNatsURL() string {
	return env.GetEnv("NATS_URL", defaultNatsURL)
}

// getNatsNodeDir returns the path to nats-node directory for task commands
func getNatsNodeDir() string {
	return env.GetEnv("NATS_NODE_DIR", defaultNatsNodeDir)
}
