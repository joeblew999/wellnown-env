package pcview

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestState_SetAndGetProcesses(t *testing.T) {
	state := NewState()

	// Initially empty
	procs, errStr := state.GetProcesses()
	assert.Empty(t, procs)
	assert.Empty(t, errStr)

	// Set processes
	testProcs := []ProcessState{
		{Name: "ticker", Status: "Running", IsRunning: true, Pid: 1234, Health: "healthy"},
		{Name: "counter", Status: "Running", IsRunning: true, Pid: 1235, Health: "healthy"},
		{Name: "logger", Status: "Disabled", IsRunning: false, Pid: 0, Health: ""},
	}
	state.SetProcesses(testProcs, "")

	// Get processes back
	procs, errStr = state.GetProcesses()
	assert.Len(t, procs, 3)
	assert.Empty(t, errStr)
	assert.Equal(t, "ticker", procs[0].Name)
	assert.Equal(t, 1234, procs[0].Pid)
	assert.True(t, procs[0].IsRunning)
}

func TestState_SetError(t *testing.T) {
	state := NewState()

	// Set with error
	state.SetProcesses(nil, "connection refused")

	procs, errStr := state.GetProcesses()
	assert.Empty(t, procs)
	assert.Equal(t, "connection refused", errStr)
}

func TestState_ConcurrentAccess(t *testing.T) {
	state := NewState()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			procs := []ProcessState{{Name: "test", Pid: n}}
			state.SetProcesses(procs, "")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have valid state (no race condition)
	procs, _ := state.GetProcesses()
	assert.Len(t, procs, 1)
}

func TestProcessState_Fields(t *testing.T) {
	proc := ProcessState{
		Name:      "ticker",
		Status:    "Running",
		IsRunning: true,
		Pid:       12345,
		Health:    "healthy",
		Restarts:  3,
	}

	assert.Equal(t, "ticker", proc.Name)
	assert.Equal(t, "Running", proc.Status)
	assert.True(t, proc.IsRunning)
	assert.Equal(t, 12345, proc.Pid)
	assert.Equal(t, "healthy", proc.Health)
	assert.Equal(t, 3, proc.Restarts)
}

func TestExampleProcesses(t *testing.T) {
	// Verify the example processes list
	assert.Contains(t, ExampleProcesses, "ticker")
	assert.Contains(t, ExampleProcesses, "counter")
	assert.Contains(t, ExampleProcesses, "logger")
	assert.Len(t, ExampleProcesses, 3)
}
