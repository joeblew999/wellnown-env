// Example: Embedding process-compose as a Go library with Via web UI
//
// This demonstrates embedding process-compose to manage processes programmatically,
// with a Via web interface for viewing and controlling processes.
//
// Environment Variables (see pc.yaml for defaults):
//   VIA_ADDR    - Via web UI bind address (default: :3000)
//   VIA_PORT    - Via web UI port (default: 3000)
//   VIA_HOST    - Via host for display URLs (default: localhost)
//   PC_ADDRESS  - Process-compose API address (default: localhost)
//   PC_PORT     - Process-compose API port (default: 8181)
//   APP_NAME    - Application name for dashboard (default: pc-node)
//   LOG_LEVEL   - Logging level (default: info)
//   DEBUG       - Enable debug mode (default: false)
//
// Run:
//
//	go run .
//
// Then open http://localhost:3000 in your browser (or VIA_URL from env)
//
// The example:
// 1. Loads a pc.yaml config
// 2. Starts all processes
// 3. Provides a Via web UI for monitoring/control
// 4. Shuts down gracefully
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/go-via/via"
	. "github.com/go-via/via/h"

	"github.com/joeblew999/wellnown-env/pkg/env"
	"github.com/joeblew999/wellnown-env/pkg/env/pcview"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Resolve any ref+ secrets in environment variables (vals)
	// This allows secrets to come from Vault, 1Password, files, etc.
	// For local dev, use ref+file://./secrets/mykey.txt
	if env.HasSecretRefs() {
		fmt.Println("Resolving secrets...")
		if err := env.ResolveEnvSecrets(); err != nil {
			return fmt.Errorf("resolving secrets: %w", err)
		}
	}

	// Load configuration from environment variables
	cfg := env.LoadConfig()

	fmt.Println("Process-Compose + Via Web UI")
	fmt.Println("============================")
	fmt.Printf("App: %s\n", cfg.AppName)
	fmt.Printf("Via: %s (bind: %s)\n", cfg.ViaURL, cfg.ViaAddr)
	fmt.Printf("PC:  %s\n", cfg.PCURL)
	if cfg.Debug {
		fmt.Printf("Log: %s (DEBUG)\n", cfg.LogLevel)
	}
	fmt.Println()

	// Step 1: Load configuration from YAML file
	fmt.Println("Loading pc.yaml...")

	loaderOpts := &loader.LoaderOptions{
		FileNames: []string{"pc.yaml"},
	}
	loaderOpts.WithTuiDisabled(true)

	project, err := loader.Load(loaderOpts)
	if err != nil {
		return fmt.Errorf("loading project: %w", err)
	}

	fmt.Printf("Loaded project with %d processes\n", len(project.Processes))
	for name := range project.Processes {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	// Step 2: Create project runner (no TUI, headless mode)
	fmt.Println("Creating project runner...")

	projectOpts := &app.ProjectOpts{}
	projectOpts.
		WithProject(project).
		WithProcessesToRun([]string{}). // Empty = run all non-disabled processes
		WithNoDeps(false).              // false means respect dependencies
		WithIsTuiOn(false).             // Headless mode
		WithOrderedShutdown(true)

	runner, err := app.NewProjectRunner(projectOpts)
	if err != nil {
		return fmt.Errorf("creating runner: %w", err)
	}

	// Step 3: Start processes in background
	fmt.Println("Starting processes...")
	fmt.Println()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run()
	}()

	// Step 4: Set up pcview with embedded runner
	pcState := pcview.NewState()

	// Create a custom client that uses the embedded runner directly
	embeddedClient := &embeddedPCClient{runner: runner}

	// Start background ticker to update state from runner
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			states, err := runner.GetProcessesState()
			if err != nil {
				pcState.SetError(err.Error())
				continue
			}
			// Convert to pcview.ProcessState
			var procs []pcview.ProcessState
			for _, s := range states.States {
				procs = append(procs, pcview.ProcessState{
					Name:      s.Name,
					Status:    string(s.Status),
					IsRunning: s.IsRunning,
					Pid:       s.Pid,
					Health:    string(s.Health),
					Restarts:  s.Restarts,
					ExitCode:  s.ExitCode,
				})
			}
			pcState.SetProcesses(procs, "")
		}
	}()

	// Step 5: Create Via web UI
	fmt.Printf("Starting Via web UI on %s\n", cfg.ViaURL)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	v := via.New()
	v.Config(via.Options{
		ServerAddress: cfg.ViaAddr,
		DocumentTitle: cfg.AppName + " Dashboard",
		LogLvl:        via.LogLevelInfo,
	})

	// Register home page
	v.Page("/", func(c *via.Context) {
		c.View(func() H {
			procs, lastErr := pcState.GetProcesses()
			var statusRows []H
			for _, p := range procs {
				status := "Stopped"
				if p.IsRunning {
					status = "Running"
				}
				health := p.Health
				if health == "" {
					health = "N/A"
				}
				statusRows = append(statusRows, Tr(
					Td(Strong(Text(p.Name))),
					Td(Text(status)),
					Td(Code(Textf("%d", p.Pid))),
					Td(Text(health)),
					Td(Textf("%d", p.Restarts)),
				))
			}

			var errEl H
			if lastErr != "" {
				errEl = P(Style("color:red"), Text("Error: "+lastErr))
			}

			return Main(Style("max-width:800px;margin:0 auto;padding:20px;font-family:system-ui"),
				H1(Text(cfg.AppName+" Dashboard")),
				P(Text("Process-Compose embedded in Go with Via web UI")),
				errEl,
				Nav(Style("margin:20px 0"),
					A(Href("/"), Text("Home")), Text(" | "),
					A(Href("/processes"), Text("Processes")), Text(" | "),
					A(Href("/examples"), Text("Examples")),
				),
				H2(Text("Process Status")),
				Table(Style("width:100%;border-collapse:collapse"),
					THead(Tr(
						Th(Text("Process")),
						Th(Text("Status")),
						Th(Text("PID")),
						Th(Text("Health")),
						Th(Text("Restarts")),
					)),
					TBody(statusRows...),
				),
				Hr(),
				P(Small(Text("Refreshes automatically via SSE"))),
			)
		})
	})

	// Shared nav bar for pcview pages
	navBar := func(title string) H {
		return Nav(Style("margin:20px 0"),
			A(Href("/"), Text("Home")), Text(" | "),
			func() H {
				if title == "Processes" {
					return Strong(Text("Processes"))
				}
				return A(Href("/processes"), Text("Processes"))
			}(),
			Text(" | "),
			func() H {
				if title == "Examples" {
					return Strong(Text("Examples"))
				}
				return A(Href("/examples"), Text("Examples"))
			}(),
		)
	}

	// Register pcview processes page with control buttons
	// Empty Controllable = all processes are controllable (default)
	pcview.RegisterPage(v, embeddedClient, pcState, pcview.PageOptions{
		NavBar: navBar,
		// Controllable: []string{"ticker", "counter"}, // Uncomment to restrict controls
	})

	// Register examples page for demo processes (regression testing)
	pcview.RegisterExamplesPage(v, embeddedClient, pcState, pcview.ExamplesPageOptions{
		NavBar: navBar,
	})

	// Start Via in background
	go v.Start()

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown or error
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("runner error: %w", err)
		}
		fmt.Println("All processes completed")
		return nil

	case <-sigCh:
		fmt.Println("\nReceived shutdown signal...")
		fmt.Println("Shutting down processes...")
		if err := runner.ShutDownProject(); err != nil {
			fmt.Printf("Shutdown error: %v\n", err)
		}
		fmt.Println("Goodbye!")
		return nil
	}
}

// embeddedPCClient implements the control interface using the embedded runner
type embeddedPCClient struct {
	runner *app.ProjectRunner
}

func (c *embeddedPCClient) GetProcesses() ([]pcview.ProcessState, error) {
	states, err := c.runner.GetProcessesState()
	if err != nil {
		return nil, err
	}
	var procs []pcview.ProcessState
	for _, s := range states.States {
		procs = append(procs, pcview.ProcessState{
			Name:      s.Name,
			Status:    string(s.Status),
			IsRunning: s.IsRunning,
			Pid:       s.Pid,
			Health:    string(s.Health),
			Restarts:  s.Restarts,
			ExitCode:  s.ExitCode,
		})
	}
	return procs, nil
}

func (c *embeddedPCClient) Control(action, name string) error {
	switch action {
	case "start":
		return c.runner.StartProcess(name)
	case "stop":
		return c.runner.StopProcess(name)
	case "restart":
		return c.runner.RestartProcess(name)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (c *embeddedPCClient) Start(name string) error   { return c.Control("start", name) }
func (c *embeddedPCClient) Stop(name string) error    { return c.Control("stop", name) }
func (c *embeddedPCClient) Restart(name string) error { return c.Control("restart", name) }
