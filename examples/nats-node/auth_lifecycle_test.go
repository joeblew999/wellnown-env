package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

// TestAuthLifecycle runs through all auth modes: none -> token -> nkey -> jwt -> none
func TestAuthLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping auth lifecycle test in short mode")
	}

	modes := []struct {
		name      string
		setupTask string
		authFunc  func() (nats.Option, error)
	}{
		{
			name:      "none",
			setupTask: "", // auth:clean sets this
			authFunc:  func() (nats.Option, error) { return nil, nil },
		},
		{
			name:      "token",
			setupTask: "auth:token",
			authFunc:  tokenAuthOption,
		},
		{
			name:      "nkey",
			setupTask: "auth:nkey",
			authFunc:  nkeyAuthOption,
		},
		{
			name:      "jwt",
			setupTask: "auth:jwt",
			authFunc:  jwtAuthOption,
		},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			testAuthMode(t, mode.name, mode.setupTask, mode.authFunc)
		})
	}

	// Final cleanup - return to none
	t.Run("cleanup", func(t *testing.T) {
		runTask(t, "clean")
		runTask(t, "auth:clean")
	})
}

func testAuthMode(t *testing.T, modeName, setupTask string, authFunc func() (nats.Option, error)) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Fresh start
	t.Logf("Fresh start for %s mode", modeName)
	runTask(t, "clean")
	runTask(t, "auth:clean")

	// Step 2: Set up auth mode
	if setupTask != "" {
		t.Logf("Setting up %s auth", modeName)
		runTask(t, setupTask)
	}

	// Step 3: Verify auth mode is set correctly
	status := getAuthStatus(t)
	if modeName == "none" {
		if status != "none" && status != "none (dev)" {
			t.Fatalf("Expected auth mode 'none', got '%s'", status)
		}
	} else {
		if status != modeName {
			t.Fatalf("Expected auth mode '%s', got '%s'", modeName, status)
		}
	}
	t.Logf("Auth mode correctly set to: %s", status)

	// Step 4: Start the hub server
	hubCmd, err := startHub(ctx, t)
	if err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}
	defer stopHub(hubCmd, t)

	// Step 5: Wait for hub to be ready
	if err := waitForHub(ctx, t); err != nil {
		t.Fatalf("Hub not ready: %v", err)
	}

	// Step 6: Test connection with correct auth
	t.Logf("Testing connection with %s auth", modeName)
	authOpt, err := authFunc()
	if err != nil {
		t.Fatalf("Failed to get auth option: %v", err)
	}

	opts := []nats.Option{nats.Timeout(5 * time.Second)}
	if authOpt != nil {
		opts = append(opts, authOpt)
	}

	nc, err := nats.Connect("nats://localhost:4222", opts...)
	if err != nil {
		t.Fatalf("Failed to connect with %s auth: %v", modeName, err)
	}
	nc.Close()
	t.Logf("Successfully connected with %s auth", modeName)

	// Step 7: Test that connection fails without auth (except for none mode)
	if modeName != "none" {
		t.Logf("Verifying connection fails without auth")
		nc, err := nats.Connect("nats://localhost:4222", nats.Timeout(2*time.Second))
		if err == nil {
			nc.Close()
			t.Fatalf("Connection should have failed without auth in %s mode", modeName)
		}
		t.Logf("Connection correctly rejected without auth")
	}
}

// runTask executes a task command
func runTask(t *testing.T, taskName string) {
	t.Helper()
	cmd := exec.Command("task", taskName)
	cmd.Dir = getProjectDir()
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Task %s output: %s", taskName, string(output))
		// Don't fail on clean tasks if nothing to clean
		if !strings.Contains(taskName, "clean") {
			t.Fatalf("Task %s failed: %v", taskName, err)
		}
	}
}

// getAuthStatus reads the current auth status
func getAuthStatus(t *testing.T) string {
	t.Helper()
	modeFile := filepath.Join(getProjectDir(), ".auth", "mode")
	data, err := os.ReadFile(modeFile)
	if err != nil {
		return "none"
	}
	return strings.TrimSpace(string(data))
}

// startHub starts the NATS hub server
func startHub(ctx context.Context, t *testing.T) (*exec.Cmd, error) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Dir = getProjectDir()
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"NATS_NAME=hub",
		"NATS_PORT=4222",
		"NATS_DATA=./.data/hub",
	)
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	t.Logf("Hub started with PID %d", cmd.Process.Pid)
	return cmd, nil
}

// stopHub stops the hub server
func stopHub(cmd *exec.Cmd, t *testing.T) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}
	t.Logf("Stopping hub (PID %d)", cmd.Process.Pid)
	cmd.Process.Kill()
	cmd.Wait()
	time.Sleep(500 * time.Millisecond)
}

// waitForHub waits for the hub to be ready
func waitForHub(ctx context.Context, t *testing.T) error {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := net.DialTimeout("tcp", "localhost:4222", 500*time.Millisecond)
		if err == nil {
			conn.Close()
			time.Sleep(500 * time.Millisecond) // Extra time for full startup
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("hub not ready after 15 seconds")
}

// getProjectDir returns the nats-node directory
func getProjectDir() string {
	// Try to find it relative to test file
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

// Auth option functions

func tokenAuthOption() (nats.Option, error) {
	tokenFile := filepath.Join(getProjectDir(), ".auth", "token")
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("reading token file: %w", err)
	}
	token := strings.TrimSpace(string(data))
	return nats.Token(token), nil
}

func nkeyAuthOption() (nats.Option, error) {
	seedFile := filepath.Join(getProjectDir(), ".auth", "user.nk")
	return nats.NkeyOptionFromSeed(seedFile), nil
}

func jwtAuthOption() (nats.Option, error) {
	credsFile := filepath.Join(getProjectDir(), ".auth", "creds", "user.creds")
	return nats.UserCredentials(credsFile), nil
}

// TestAuthModeTransitions tests direct transitions between specific modes
func TestAuthModeTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping auth mode transitions test in short mode")
	}

	transitions := []struct {
		from string
		to   string
	}{
		{"none", "token"},
		{"token", "nkey"},
		{"nkey", "jwt"},
		{"jwt", "none"},
	}

	for _, tr := range transitions {
		t.Run(fmt.Sprintf("%s_to_%s", tr.from, tr.to), func(t *testing.T) {
			// Clean and set starting mode
			runTask(t, "clean")
			runTask(t, "auth:clean")

			if tr.from != "none" {
				runTask(t, "auth:"+tr.from)
			}

			// Verify starting mode
			status := getAuthStatus(t)
			if tr.from == "none" {
				if status != "none" && status != "" {
					t.Logf("Warning: expected 'none', got '%s'", status)
				}
			} else if status != tr.from {
				t.Fatalf("Starting mode should be %s, got %s", tr.from, status)
			}

			// Clean data (simulating fresh start)
			runTask(t, "clean")

			// Transition to new mode
			if tr.to == "none" {
				runTask(t, "auth:clean")
			} else {
				runTask(t, "auth:clean") // Clean old auth first
				runTask(t, "auth:"+tr.to)
			}

			// Verify new mode
			status = getAuthStatus(t)
			if tr.to == "none" {
				if status != "none" && status != "" {
					t.Fatalf("Expected 'none' after transition, got '%s'", status)
				}
			} else if status != tr.to {
				t.Fatalf("Expected '%s' after transition, got '%s'", tr.to, status)
			}

			t.Logf("Successfully transitioned from %s to %s", tr.from, tr.to)
		})
	}

	// Final cleanup
	runTask(t, "clean")
	runTask(t, "auth:clean")
}
