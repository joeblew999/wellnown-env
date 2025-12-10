package pcview

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockPCServer creates a mock process-compose API server for testing
func mockPCServer(t *testing.T, processes []ProcessState) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/processes":
			resp := ProcessStates{States: processes}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case r.Method == "POST" && len(r.URL.Path) > 9 && r.URL.Path[:9] == "/process/":
			// Handle /process/{action}/{name}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestClient_GetProcesses(t *testing.T) {
	// Create mock server
	server := mockPCServer(t, []ProcessState{
		{Name: "ticker", Status: "Running", IsRunning: true, Pid: 1234, Health: "healthy"},
		{Name: "counter", Status: "Running", IsRunning: true, Pid: 1235, Health: "healthy"},
	})
	defer server.Close()

	// Create client
	client := NewClient(server.URL)

	// Get processes
	procs, err := client.GetProcesses()
	assert.NoError(t, err)
	assert.Len(t, procs, 2)
	assert.Equal(t, "ticker", procs[0].Name)
	assert.Equal(t, 1234, procs[0].Pid)
}

func TestClient_Control(t *testing.T) {
	server := mockPCServer(t, nil)
	defer server.Close()

	client := NewClient(server.URL)

	// Test control actions
	assert.NoError(t, client.Control("start", "ticker"))
	assert.NoError(t, client.Control("stop", "ticker"))
	assert.NoError(t, client.Control("restart", "ticker"))
}

func TestClient_StartStopRestart(t *testing.T) {
	server := mockPCServer(t, nil)
	defer server.Close()

	client := NewClient(server.URL)

	assert.NoError(t, client.Start("ticker"))
	assert.NoError(t, client.Stop("ticker"))
	assert.NoError(t, client.Restart("counter"))
}

func TestClient_ConnectionError(t *testing.T) {
	// Create client with invalid URL
	client := NewClient("http://localhost:99999")

	// Should return error
	_, err := client.GetProcesses()
	assert.Error(t, err)
}

func TestClient_ImplementsProcessController(t *testing.T) {
	// Verify Client implements ProcessController interface
	var _ ProcessController = (*Client)(nil)
}

// MockController is a test implementation of ProcessController
type MockController struct {
	processes []ProcessState
	actions   []string
}

func (m *MockController) GetProcesses() ([]ProcessState, error) {
	return m.processes, nil
}

func (m *MockController) Control(action, name string) error {
	m.actions = append(m.actions, action+":"+name)
	return nil
}

func (m *MockController) Start(name string) error {
	return m.Control("start", name)
}

func (m *MockController) Stop(name string) error {
	return m.Control("stop", name)
}

func (m *MockController) Restart(name string) error {
	return m.Control("restart", name)
}

func TestMockController(t *testing.T) {
	// MockController can be used for testing Via pages
	mock := &MockController{
		processes: []ProcessState{
			{Name: "ticker", IsRunning: true},
			{Name: "counter", IsRunning: false},
		},
	}

	// Test interface compliance
	var controller ProcessController = mock

	procs, err := controller.GetProcesses()
	assert.NoError(t, err)
	assert.Len(t, procs, 2)

	controller.Start("counter")
	controller.Stop("ticker")

	assert.Contains(t, mock.actions, "start:counter")
	assert.Contains(t, mock.actions, "stop:ticker")
}
