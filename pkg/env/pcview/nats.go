package pcview

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSHandler provides NATS request/response handlers for process-compose
type NATSHandler struct {
	client   *Client
	state    *State
	nc       *nats.Conn
	onUpdate func() // Called when state changes
}

// NewNATSHandler creates a new NATS handler
func NewNATSHandler(client *Client, state *State, nc *nats.Conn) *NATSHandler {
	return &NATSHandler{
		client: client,
		state:  state,
		nc:     nc,
	}
}

// OnUpdate sets a callback for when process state changes
func (h *NATSHandler) OnUpdate(fn func()) {
	h.onUpdate = fn
}

// StartStatusResponder subscribes to pc.processes and responds with current state
// This should run on the node that has direct access to process-compose API
func (h *NATSHandler) StartStatusResponder() error {
	_, err := h.nc.Subscribe(SubjectStatus, func(msg *nats.Msg) {
		states, err := h.client.GetProcesses()
		if err != nil {
			_ = msg.Respond([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
			return
		}
		body, _ := json.Marshal(states)
		_ = msg.Respond(body)
	})
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", SubjectStatus, err)
	}
	return nil
}

// StartControlResponder subscribes to pc.processes.control and proxies to process-compose
// This should run on the node that has direct access to process-compose API
func (h *NATSHandler) StartControlResponder() error {
	_, err := h.nc.Subscribe(SubjectControl, func(msg *nats.Msg) {
		var req ControlRequest
		respond := func(ok bool, errMsg string) {
			resp := ControlResponse{OK: ok, Error: errMsg}
			body, _ := json.Marshal(resp)
			_ = msg.Respond(body)
		}

		if err := json.Unmarshal(msg.Data, &req); err != nil {
			respond(false, "bad request")
			return
		}
		if req.Action == "" || req.Name == "" {
			respond(false, "action and name required")
			return
		}

		if err := h.client.Control(req.Action, req.Name); err != nil {
			respond(false, err.Error())
		} else {
			respond(true, "")
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", SubjectControl, err)
	}
	return nil
}

// StartUpdatesSubscription subscribes to pc.processes.updates and updates local state
// This should run on Via nodes that display process state
func (h *NATSHandler) StartUpdatesSubscription() error {
	sub, err := h.nc.Subscribe(SubjectUpdates, func(msg *nats.Msg) {
		var states []ProcessState
		if err := json.Unmarshal(msg.Data, &states); err != nil {
			h.state.SetError(err.Error())
		} else {
			h.state.SetProcesses(states, "")
		}
		if h.onUpdate != nil {
			h.onUpdate()
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", SubjectUpdates, err)
	}

	h.state.mu.Lock()
	h.state.updatesSub = sub
	h.state.mu.Unlock()
	return nil
}

// PublishUpdate broadcasts current process state to all subscribers
func (h *NATSHandler) PublishUpdate(states []ProcessState) error {
	body, err := json.Marshal(states)
	if err != nil {
		return fmt.Errorf("marshal states: %w", err)
	}
	return h.nc.Publish(SubjectUpdates, body)
}

// RequestProcesses sends a request to get current process state via NATS
func (h *NATSHandler) RequestProcesses() ([]ProcessState, error) {
	resp, err := h.nc.Request(SubjectStatus, nil, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("request processes: %w", err)
	}

	// Check for error response
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(resp.Data, &errResp) == nil && errResp.Error != "" {
		return nil, fmt.Errorf("%s", errResp.Error)
	}

	var states []ProcessState
	if err := json.Unmarshal(resp.Data, &states); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return states, nil
}

// ControlViaNATS sends a control command via NATS
func (h *NATSHandler) ControlViaNATS(action, name string) error {
	req := ControlRequest{Action: action, Name: name}
	body, _ := json.Marshal(req)

	resp, err := h.nc.Request(SubjectControl, body, 3*time.Second)
	if err != nil {
		return fmt.Errorf("control request: %w", err)
	}

	var out ControlResponse
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !out.OK {
		if out.Error != "" {
			return fmt.Errorf("%s", out.Error)
		}
		return fmt.Errorf("control failed")
	}
	return nil
}
