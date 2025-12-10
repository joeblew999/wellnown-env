package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// TaskResult holds the result of running a task command
type TaskResult struct {
	Command  string
	Output   string
	Error    string
	ExitCode int
}

var (
	taskMu        sync.Mutex
	lastResult    TaskResult
	lastResultMu  sync.RWMutex
)

// RunTask executes a task command in the nats-node directory
func RunTask(taskName string) TaskResult {
	taskMu.Lock()
	defer taskMu.Unlock()

	result := TaskResult{
		Command: taskName,
	}

	natsNodeDir := getNatsNodeDir()

	// Resolve to absolute path
	absDir, err := filepath.Abs(natsNodeDir)
	if err != nil {
		result.Error = "Failed to resolve nats-node directory: " + err.Error()
		result.ExitCode = 1
		setLastResult(result)
		return result
	}

	cmd := exec.Command("task", taskName)
	cmd.Dir = absDir
	cmd.Env = append(os.Environ(), "GOWORK=off")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result.Output = stdout.String()

	if err != nil {
		result.Error = stderr.String()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.ExitCode = 0
	}

	setLastResult(result)
	return result
}

// setLastResult stores the last task result
func setLastResult(r TaskResult) {
	lastResultMu.Lock()
	defer lastResultMu.Unlock()
	lastResult = r
}

// GetLastResult returns the last task result
func GetLastResult() TaskResult {
	lastResultMu.RLock()
	defer lastResultMu.RUnlock()
	return lastResult
}

// GetAuthStatus reads the current auth mode from .auth/mode file
func GetAuthStatus() string {
	natsNodeDir := getNatsNodeDir()
	modeFile := filepath.Join(natsNodeDir, ".auth", "mode")

	data, err := os.ReadFile(modeFile)
	if err != nil {
		return "none"
	}

	return strings.TrimSpace(string(data))
}

// GetAuthToken reads the token if in token mode
func GetAuthToken() string {
	natsNodeDir := getNatsNodeDir()
	tokenFile := filepath.Join(natsNodeDir, ".auth", "token")

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

// GetNKeyPub reads the NKey public key if in nkey mode
func GetNKeyPub() string {
	natsNodeDir := getNatsNodeDir()
	pubFile := filepath.Join(natsNodeDir, ".auth", "user.pub")

	data, err := os.ReadFile(pubFile)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

// IsMeshRunning checks if the mesh is currently running
func IsMeshRunning() bool {
	result := RunTask("mesh:list")
	// If mesh:list returns output with process names, mesh is running
	return result.ExitCode == 0 && strings.Contains(result.Output, "hub")
}
