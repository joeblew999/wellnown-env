package pcview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/joeblew999/wellnown-env/pkg/env"
)

// Client talks to the process-compose HTTP API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new process-compose API client with the given base URL
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewClientFromEnv creates a new process-compose API client using environment variables.
// Uses PC_URL, PC_ADDRESS, and PC_PORT to construct the base URL.
func NewClientFromEnv() *Client {
	return NewClient(env.GetProcessComposeURL())
}

// GetProcesses fetches current process states from process-compose API
func (c *Client) GetProcesses() ([]ProcessState, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/processes")
	if err != nil {
		return nil, fmt.Errorf("fetch processes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var states ProcessStates
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, fmt.Errorf("decode processes: %w", err)
	}
	return states.States, nil
}

// Control sends a control command (start/stop/restart) to a process
func (c *Client) Control(action, name string) error {
	url := fmt.Sprintf("%s/process/%s/%s", c.baseURL, action, name)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("control %s %s: %w", action, name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return nil
}

// Start starts a process
func (c *Client) Start(name string) error {
	return c.Control("start", name)
}

// Stop stops a process
func (c *Client) Stop(name string) error {
	return c.Control("stop", name)
}

// Restart restarts a process
func (c *Client) Restart(name string) error {
	return c.Control("restart", name)
}
