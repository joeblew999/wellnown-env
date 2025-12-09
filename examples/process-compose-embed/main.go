// Example: Embedding process-compose as a Go library
//
// This demonstrates embedding process-compose to manage processes programmatically,
// similar to how we embed NATS. This could be used to manage binaries pulled via NATS.
//
// Run:
//   go run main.go
//
// The example:
// 1. Loads a process-compose.yaml config
// 2. Starts all processes
// 3. Monitors their status
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
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("Process-Compose Embedded Example")
	fmt.Println("=================================")
	fmt.Println()

	// Step 1: Load configuration from YAML file
	fmt.Println("Loading process-compose.yaml...")

	loaderOpts := &loader.LoaderOptions{
		FileNames: []string{"process-compose.yaml"},
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
		WithProcessesToRun([]string{}).    // Empty = run all non-disabled processes
		WithNoDeps(false).                  // false means respect dependencies
		WithIsTuiOn(false).                 // Headless mode
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

	// Step 4: Monitor process status
	fmt.Println("Monitoring processes (Ctrl+C to stop)...")
	fmt.Println()

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
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

		case <-ticker.C:
			// Get all process states
			states, err := runner.GetProcessesState()
			if err != nil {
				fmt.Printf("Error getting states: %v\n", err)
				continue
			}

			fmt.Println("--- Process Status ---")
			for _, state := range states.States {
				status := state.Status
				if state.IsRunning {
					status = "Running"
				}
				health := state.Health
				if health == "" {
					health = "N/A"
				}
				fmt.Printf("  [%s] Status: %s, PID: %d, Health: %s, Restarts: %d\n",
					state.Name, status, state.Pid, health, state.Restarts)
			}
			fmt.Println("----------------------")
			fmt.Println()

			// Check if all ready (for services with health probes)
			if states.IsReady() {
				fmt.Println("All processes are ready!")
			}
		}
	}
}
