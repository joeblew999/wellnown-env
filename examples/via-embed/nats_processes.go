package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	processStatusSubject  = "pc.processes"
	processControlSubject = "pc.processes.control"
	processUpdatesSubject = "pc.processes.updates"
)

// updateProcessCache updates shared cache and optionally broadcasts
func updateProcessCache(states []ProcessState, err error, notify bool) {
	processesNATSMu.Lock()
	if err != nil {
		processesNATSError = err.Error()
		processesNATS = nil
	} else {
		processesNATSError = ""
		processesNATS = states
	}
	processesNATSMu.Unlock()
	if notify {
		broadcast.Notify(TopicProcessesNATS)
	}
}

// startProcessStatusResponder replies to NATS requests with current process-compose state
func startProcessStatusResponder(ctx context.Context) error {
	nc, err := getNatsConn()
	if err != nil {
		return err
	}

	pcURL := getProcessComposeURL()
	_, err = nc.Subscribe(processStatusSubject, func(msg *nats.Msg) {
		states, err := fetchProcessStates(pcURL)
		if err != nil {
			_ = msg.Respond([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
			return
		}
		body, _ := json.Marshal(states)
		_ = msg.Respond(body)
	})
	if err != nil {
		return fmt.Errorf("subscribe process status: %w", err)
	}
	return nil
}

// getProcessesViaNATS requests process state via NATS
func getProcessesViaNATS() ([]ProcessState, error) {
	nc, err := getNatsConn()
	if err != nil {
		return nil, err
	}
	resp, err := nc.Request(processStatusSubject, nil, 2*time.Second)
	if err != nil {
		return nil, err
	}
	var states []ProcessState
	if err := json.Unmarshal(resp.Data, &states); err != nil {
		return nil, fmt.Errorf("decode process state: %w", err)
	}
	return states, nil
}

// requestProcessesViaNATS fetches process state and updates shared cache
func requestProcessesViaNATS() ([]ProcessState, error) {
	states, err := getProcessesViaNATS()
	updateProcessCache(states, err, false)
	return states, err
}

// publishProcessUpdate broadcasts process state over NATS
func publishProcessUpdate(states []ProcessState) {
	nc, err := getNatsConn()
	if err != nil {
		return
	}
	body, err := json.Marshal(states)
	if err != nil {
		return
	}
	_ = nc.Publish(processUpdatesSubject, body)
}

// startProcessUpdatesSubscription listens for pc.processes.updates and updates liveProcesses cache
func startProcessUpdatesSubscription() error {
	nc, err := getNatsConn()
	if err != nil {
		return err
	}

	processesNATSMu.Lock()
	if processesUpdatesSub != nil {
		_ = processesUpdatesSub.Unsubscribe()
		processesUpdatesSub = nil
	}
	processesNATSMu.Unlock()

	sub, err := nc.Subscribe(processUpdatesSubject, func(msg *nats.Msg) {
		var states []ProcessState
		if err := json.Unmarshal(msg.Data, &states); err != nil {
			processesMu.Lock()
			processesError = err.Error()
			liveProcesses = nil
			processesMu.Unlock()
			broadcast.Notify(TopicProcesses)
			return
		}
		// Update the main liveProcesses cache (used by /processes page)
		processesMu.Lock()
		processesError = ""
		liveProcesses = states
		processesMu.Unlock()
		broadcast.Notify(TopicProcesses)
	})
	if err != nil {
		return fmt.Errorf("subscribe process updates: %w", err)
	}

	processesNATSMu.Lock()
	processesUpdatesSub = sub
	processesNATSMu.Unlock()
	return nil
}

// controlProcessViaNATS sends start/stop/restart via NATS
func controlProcessViaNATS(action, name string) error {
	nc, err := getNatsConn()
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"action": action, "name": name})
	resp, err := nc.Request(processControlSubject, body, 3*time.Second)
	if err != nil {
		return err
	}
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(resp.Data, &out)
	if !out.OK {
		if out.Error != "" {
			return fmt.Errorf("%s", out.Error)
		}
		return fmt.Errorf("control failed")
	}
	return nil
}

// startProcessControlResponder listens for control requests and proxies to process-compose HTTP API
func startProcessControlResponder(_ context.Context) error {
	nc, err := getNatsConn()
	if err != nil {
		return err
	}

	pcURL := getProcessComposeURL()
	client := &http.Client{Timeout: 10 * time.Second}

	_, err = nc.Subscribe(processControlSubject, func(msg *nats.Msg) {
		var req struct {
			Action string `json:"action"`
			Name   string `json:"name"`
		}
		respond := func(ok bool, errMsg string) {
			resp := struct {
				OK    bool   `json:"ok"`
				Error string `json:"error,omitempty"`
			}{OK: ok, Error: errMsg}
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

		err := controlProcessWithClient(client, pcURL, req.Action, req.Name)
		if err != nil {
			respond(false, err.Error())
		} else {
			respond(true, "")
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe process control: %w", err)
	}
	return nil
}

// controlProcessWithClient sends a control command to process-compose API
func controlProcessWithClient(client *http.Client, pcURL, action, name string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/process/%s/%s", pcURL, action, name), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return nil
}
