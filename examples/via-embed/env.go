package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-via/via-plugin-picocss/picocss"
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

// themeMap maps theme names to picocss.Theme constants
var themeMap = map[string]picocss.Theme{
	"amber":   picocss.ThemeAmber,
	"blue":    picocss.ThemeBlue,
	"cyan":    picocss.ThemeCyan,
	"fuchsia": picocss.ThemeFuchia,
	"green":   picocss.ThemeGreen,
	"grey":    picocss.ThemeGrey,
	"indigo":  picocss.ThemeIndigo,
	"jade":    picocss.ThemeJade,
	"lime":    picocss.ThemeLime,
	"orange":  picocss.ThemeOrange,
	"pink":    picocss.ThemePink,
	"pumpkin": picocss.ThemePumpkin,
	"purple":  picocss.ThemePurple,
	"red":     picocss.ThemeRed,
	"sand":    picocss.ThemeSand,
	"slate":   picocss.ThemeSlate,
	"violet":  picocss.ThemeViolet,
	"yellow":  picocss.ThemeYellow,
	"zinc":    picocss.ThemeZinc,
}

var themeNames = []string{
	"amber", "blue", "cyan", "fuchsia", "green",
	"grey", "indigo", "jade", "lime", "orange",
	"pink", "pumpkin", "purple", "red", "sand",
	"slate", "violet", "yellow", "zinc",
}

// getThemeFromEnv reads VIA_THEME env var and returns the corresponding theme
func getThemeFromEnv() (picocss.Theme, string) {
	themeName := strings.ToLower(getEnv("VIA_THEME", defaultViaTheme))
	if theme, ok := themeMap[themeName]; ok {
		return theme, themeName
	}
	// Default to purple if invalid theme name (matches .env)
	return picocss.ThemePurple, defaultViaTheme
}

// getViaPort returns the Via server port (VIA_PORT env or default)
func getViaPort() string {
	return getEnv("VIA_PORT", defaultViaPort)
}

// getViaAddress returns the Via server address in :port format
func getViaAddress() string {
	return ":" + getViaPort()
}

// getProcessComposeURL constructs URL from PC_ADDRESS and PC_PORT env vars
func getProcessComposeURL() string {
	// Allow full URL override for flexibility
	if url := os.Getenv("PC_URL"); url != "" {
		return url
	}
	addr := getEnv("PC_ADDRESS", defaultPCAddress)
	port := getEnv("PC_PORT", defaultPCPort)
	return fmt.Sprintf("http://%s:%s", addr, port)
}

// getNatsURL returns NATS server URL (NATS_URL env or default)
func getNatsURL() string {
	return getEnv("NATS_URL", defaultNatsURL)
}

// getNatsKVBucket returns the NATS KV bucket name for config (NATS_KV_CONFIG env or default)
func getNatsKVBucket() string {
	return getEnv("NATS_KV_CONFIG", defaultNatsKVConfig)
}

// getEnv returns the value of an environment variable or a default
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
