// Package pcview provides process-compose viewing and control via Via and NATS.
package pcview

import (
	"sync"

	"github.com/nats-io/nats.go"
)

// NATS subjects for process-compose communication
const (
	SubjectStatus  = "pc.processes"
	SubjectControl = "pc.processes.control"
	SubjectUpdates = "pc.processes.updates"
)

// ProcessState represents a single process from process-compose
type ProcessState struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	IsRunning bool   `json:"is_running"`
	Pid       int    `json:"pid"`
	Health    string `json:"health"`
	Restarts  int    `json:"restarts"`
	ExitCode  int    `json:"exit_code"`
}

// ProcessStates is the response from process-compose /processes endpoint
type ProcessStates struct {
	States []ProcessState `json:"data"`
}

// ControlRequest is sent via NATS to control a process
type ControlRequest struct {
	Action string `json:"action"` // start, stop, restart
	Name   string `json:"name"`
}

// ControlResponse is the reply from a control request
type ControlResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// State holds the shared state for process viewing
type State struct {
	mu         sync.RWMutex
	processes  []ProcessState
	lastError  string
	updatesSub *nats.Subscription
}

// NewState creates a new State
func NewState() *State {
	return &State{}
}

// GetProcesses returns a copy of the current process states
func (s *State) GetProcesses() ([]ProcessState, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	procs := make([]ProcessState, len(s.processes))
	copy(procs, s.processes)
	return procs, s.lastError
}

// SetProcesses updates the process states
func (s *State) SetProcesses(procs []ProcessState, err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processes = procs
	s.lastError = err
}

// SetError sets an error message
func (s *State) SetError(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err
}

// ClearError clears the error message
func (s *State) ClearError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = ""
}
