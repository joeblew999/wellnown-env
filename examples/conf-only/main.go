// Example: Using ardanlabs/conf for struct-based configuration
//
// This demonstrates the foundation that wellnown-env builds upon:
// - Struct tags define config schema
// - Environment variables are automatically mapped
// - CLI flags are generated
// - Help text is auto-generated
//
// Run:
//   go run main.go --help
//   APP_SERVER_HOST=0.0.0.0:9000 go run main.go
//   go run main.go --server-host=0.0.0.0:9000
package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ardanlabs/conf/v3"
)

// Config defines what this service needs.
// This struct IS the schema - no separate manifest needed.
type Config struct {
	conf.Version
	Server struct {
		Host         string        `conf:"default:0.0.0.0:8080,help:Host address to bind to"`
		ReadTimeout  time.Duration `conf:"default:5s,help:Read timeout"`
		WriteTimeout time.Duration `conf:"default:10s,help:Write timeout"`
		ShutdownWait time.Duration `conf:"default:30s,short:w,help:Graceful shutdown wait"`
	}
	DB struct {
		Host     string `conf:"default:localhost:5432,help:Database host"`
		User     string `conf:"default:postgres,help:Database user"`
		Password string `conf:"mask,required,help:Database password"` // mask hides in help output
		Name     string `conf:"default:myapp,help:Database name"`
		MaxConns int    `conf:"default:10,help:Max DB connections"`
	}
	// Custom env var name override
	APIKey string `conf:"env:MY_API_KEY,mask,help:API key (uses MY_API_KEY env var)"`
	// noprint - won't show in conf.String() output
	InternalSecret string `conf:"noprint,help:Internal secret (hidden from output)"`
	Debug          bool   `conf:"default:false,short:d,help:Enable debug mode"`
}

// Build info - set via ldflags
var build = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var cfg Config
	cfg.Version.Build = build

	// Parse config from:
	// 1. Environment variables (APP_SERVER_HOST, APP_DB_PASSWORD, etc.)
	// 2. Command line flags (--server-host, --db-password, etc.)
	//
	// The prefix "APP" means env vars are prefixed with APP_
	help, err := conf.Parse("APP", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// Show what was parsed
	fmt.Println("Configuration loaded:")
	fmt.Println("=====================")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config output: %w", err)
	}
	fmt.Println(out)

	fmt.Println("\nService would start with:")
	fmt.Printf("  Server: %s\n", cfg.Server.Host)
	fmt.Printf("  DB:     %s@%s/%s\n", cfg.DB.User, cfg.DB.Host, cfg.DB.Name)
	fmt.Printf("  Debug:  %v\n", cfg.Debug)

	return nil
}
