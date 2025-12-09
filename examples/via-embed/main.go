// Example: Embedding Via for reactive web UIs in pure Go
//
// Via lets you build full-featured web applications entirely in Go:
// - No JavaScript, no templates, no transpilation
// - Real-time reactivity via Server-Sent Events (SSE)
// - Type-safe UI composition with the h package
//
// FUNKY FEATURES:
// - Connects to NATS hub for live config sync
// - Theme changes propagate to ALL Via instances via NATS KV
// - /services page shows live service registry
// - /chat page for cross-instance messaging
//
// Run:
//
//	go run .
//
// Change theme via environment variable:
//
//	VIA_THEME=purple go run .
//	VIA_THEME=amber go run .
//
// Then open http://localhost:3000 in your browser
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via-plugin-picocss/picocss"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("Via Embedded Example + NATS Integration")
	fmt.Println("========================================")
	fmt.Println()

	// Get theme from environment
	theme, themeName := getThemeFromEnv()
	fmt.Printf("Using theme: %s (set VIA_THEME env to change)\n\n", themeName)

	// Create context for background tasks
	ctx := context.Background()

	// Try to connect to NATS (non-blocking)
	go func() {
		for {
			if err := connectToNATS(); err != nil {
				fmt.Printf("NATS connection failed: %v (retrying in 5s...)\n", err)
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}
	}()

	// Start background tickers for non-NATS data sources (time, processes)
	startBackgroundTickers(ctx)

	v := via.New()

	v.Config(via.Options{
		ServerAddress: getViaAddress(),
		DocumentTitle: "wellnown-env Dashboard",
		LogLvl:        via.LogLevelInfo,
		Plugins: []via.Plugin{
			picocss.WithOptions(picocss.Options{
				Theme:         theme,
				IncludeColors: true,
			}),
		},
	})

	// Register all page handlers
	registerHomePage(v)
	registerCounterPage(v)
	registerMonitorPage(v)
	registerConfigPage(v)
	registerServicesPage(v)
	registerChatPage(v)
	registerThemesPage(v)
	registerVersionPage(v)
	registerRTLPage(v)
	registerProcessesPage(v)
	registerLivePage(v)
	registerHealthzPage(v)

	fmt.Printf("Starting Via server on http://localhost:%s\n", getViaPort())
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	v.Start()

	return nil
}
