// Package env provides shared environment variable utilities for wellnown-env projects.
//
// This package consolidates common patterns used across examples:
//   - GetEnv/GetEnvInt/GetEnvBool: Read environment variables with defaults
//   - GetProcessComposeURL: Build PC API URL from env vars
//   - GetViaAddr: Get Via server address from env vars
//
// Environment Variables:
//
//	Process-Compose:
//	  PC_URL      - Full PC API URL (overrides PC_ADDRESS+PC_PORT)
//	  PC_ADDRESS  - PC API address (default: localhost)
//	  PC_PORT     - PC API port (default: 8181)
//
//	Via Web UI:
//	  VIA_ADDR    - Via server bind address (default: :3000)
//	  VIA_PORT    - Via server port (default: 3000)
//	  VIA_HOST    - Via server host for URLs (default: localhost)
//
//	NATS:
//	  NATS_NAME   - Node name
//	  NATS_PORT   - NATS client port
//	  NATS_HUB    - Hub URL for leaf nodes
//	  NATS_DATA   - Data directory
//
// Usage:
//
//	import "github.com/joeblew999/wellnown-env/pkg/env"
//
//	port := env.GetEnv("PORT", "8080")
//	count := env.GetEnvInt("COUNT", 10)
//	pcURL := env.GetProcessComposeURL()
//	viaAddr := env.GetViaAddr()
package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Default values for process-compose
const (
	DefaultPCAddress = "localhost"
	DefaultPCPort    = "8181"
)

// Default values for Via web UI
const (
	DefaultViaHost = "localhost"
	DefaultViaPort = "3000"
	DefaultViaAddr = ":3000"
)

// GetEnv returns the value of an environment variable or a default.
func GetEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// GetEnvInt returns the value of an environment variable as int or a default.
func GetEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

// GetProcessComposeURL constructs the process-compose API URL from env vars.
// Checks PC_URL first (full override), then builds from PC_ADDRESS and PC_PORT.
func GetProcessComposeURL() string {
	if url := os.Getenv("PC_URL"); url != "" {
		return url
	}
	addr := GetEnv("PC_ADDRESS", DefaultPCAddress)
	port := GetEnv("PC_PORT", DefaultPCPort)
	return fmt.Sprintf("http://%s:%s", addr, port)
}

// GetEnvBool returns the value of an environment variable as bool or a default.
// Accepts "true", "1", "yes" (case-insensitive) as true values.
func GetEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		v = strings.ToLower(v)
		return v == "true" || v == "1" || v == "yes"
	}
	return defaultVal
}

// GetViaAddr returns the Via server bind address from env vars.
// Checks VIA_ADDR first, then builds from VIA_PORT.
func GetViaAddr() string {
	if addr := os.Getenv("VIA_ADDR"); addr != "" {
		return addr
	}
	port := GetEnv("VIA_PORT", DefaultViaPort)
	return ":" + port
}

// GetViaURL returns the full Via server URL for display/linking.
// Builds from VIA_HOST and VIA_PORT.
func GetViaURL() string {
	host := GetEnv("VIA_HOST", DefaultViaHost)
	port := GetEnv("VIA_PORT", DefaultViaPort)
	return fmt.Sprintf("http://%s:%s", host, port)
}

// MustGetEnv returns the value of an environment variable or panics if not set.
func MustGetEnv(key string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	panic(fmt.Sprintf("required environment variable %s is not set", key))
}

// Config holds all environment configuration for a wellnown-env application.
// Use LoadConfig() to populate from environment variables.
type Config struct {
	// Process-Compose settings
	PCAddress string // PC_ADDRESS - API address (default: localhost)
	PCPort    string // PC_PORT - API port (default: 8181)
	PCURL     string // Computed: http://PCAddress:PCPort

	// Via Web UI settings
	ViaAddr string // VIA_ADDR - bind address (default: :3000)
	ViaHost string // VIA_HOST - host for URLs (default: localhost)
	ViaPort string // VIA_PORT - port (default: 3000)
	ViaURL  string // Computed: http://ViaHost:ViaPort

	// Application settings
	AppName  string // APP_NAME - application name
	LogLevel string // LOG_LEVEL - logging level (default: info)
	Debug    bool   // DEBUG - enable debug mode
}

// LoadConfig reads environment variables and returns a Config struct.
func LoadConfig() *Config {
	cfg := &Config{
		PCAddress: GetEnv("PC_ADDRESS", DefaultPCAddress),
		PCPort:    GetEnv("PC_PORT", DefaultPCPort),
		ViaHost:   GetEnv("VIA_HOST", DefaultViaHost),
		ViaPort:   GetEnv("VIA_PORT", DefaultViaPort),
		AppName:   GetEnv("APP_NAME", "wellnown-env"),
		LogLevel:  GetEnv("LOG_LEVEL", "info"),
		Debug:     GetEnvBool("DEBUG", false),
	}

	// Computed values
	cfg.PCURL = GetProcessComposeURL()
	cfg.ViaAddr = GetViaAddr()
	cfg.ViaURL = GetViaURL()

	return cfg
}
