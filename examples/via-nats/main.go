// via-nats: Web GUI for NATS Auth Lifecycle Management
//
// This Via dashboard provides web-based management of:
// - Auth lifecycle (none → token → nkey → jwt)
// - Mesh control (start/stop via process-compose)
// - Testing operations (account info, KV, services)
// - Service registry viewing
//
// Run:
//
//	cd examples/nats-node && task mesh  # Start NATS mesh first
//	cd examples/via-nats && go run .    # Start this dashboard
//
// Then open http://localhost:3001 in your browser
package main

import (
	"fmt"
	"os"

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
	fmt.Println("Via NATS Auth Lifecycle Dashboard")
	fmt.Println("==================================")
	fmt.Println()

	// Get theme from environment
	theme, themeName := getThemeFromEnv()
	fmt.Printf("Using theme: %s (set VIA_THEME env to change)\n", themeName)
	fmt.Printf("NATS Node Dir: %s\n\n", getNatsNodeDir())

	v := via.New()

	v.Config(via.Options{
		ServerAddress: getViaAddress(),
		DocumentTitle: "NATS Auth Lifecycle",
		LogLvl:        via.LogLevelInfo,
		Plugins: []via.Plugin{
			picocss.WithOptions(picocss.Options{
				Theme:         theme,
				IncludeColors: true,
			}),
		},
	})

	// Register page handlers
	registerDashboardPage(v)
	registerAuthPage(v)
	registerMeshPage(v)
	registerTestsPage(v)

	fmt.Printf("Starting Via server on http://localhost:%s\n", getViaPort())
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	v.Start()

	return nil
}
