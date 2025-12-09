package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// fetchProcessStates calls process-compose API to get process states
func fetchProcessStates(pcURL string) ([]ProcessState, error) {
	resp, err := http.Get(pcURL + "/processes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var states ProcessStates
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, err
	}
	return states.States, nil
}

// controlProcess sends a control command to process-compose API
func controlProcess(pcURL, action, name string) error {
	client := &http.Client{Timeout: 10 * time.Second}
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

func registerProcessesPage(v *via.V) {
	v.Page("/processes", func(c *via.Context) {
		pcURL := getProcessComposeURL()
		var lastAction string

		// Helper to create control actions (returns OnClick H)
		// Note: Process state updates come from NATS hub (nats-embedded polls process-compose)
		makeControl := func(action, name, msg string) H {
			return c.Action(func() {
				if err := controlProcess(pcURL, action, name); err != nil {
					processesMu.Lock()
					processesError = err.Error()
					processesMu.Unlock()
				} else {
					lastAction = msg
				}
				c.Sync()
				// State update will arrive via NATS subscription within 2 seconds
			}).OnClick()
		}

		// Refresh is now a no-op since data comes via NATS subscription
		// The button just syncs the current view
		refresh := c.Action(func() {
			c.Sync()
		})

		// Process control actions (store OnClick H values)
		actions := map[string]map[string]H{
			"nats": {
				"stop":    makeControl("stop", "nats", "Stopped nats"),
				"start":   makeControl("start", "nats", "Started nats"),
				"restart": makeControl("restart", "nats", "Restarted nats"),
			},
			"via": {
				"stop":  makeControl("stop", "via", "Stopped via (page will disconnect)"),
				"start": makeControl("start", "via", "Started via"),
				"restart": c.Action(func() {
					lastAction = "Restarting via... (page will reconnect)"
					c.Sync()
					time.Sleep(100 * time.Millisecond)
					if err := controlProcess(pcURL, "restart", "via"); err != nil {
						processesMu.Lock()
						processesError = err.Error()
						processesMu.Unlock()
						c.Sync()
					}
				}).OnClick(),
			},
		}

		broadcast.Subscribe(TopicProcesses, func() {
			lastAction = ""
			c.Sync()
		})

		c.View(func() H {
			processesMu.RLock()
			processes := make([]ProcessState, len(liveProcesses))
			copy(processes, liveProcesses)
			lastError := processesError
			processesMu.RUnlock()

			var rows []H
			for _, proc := range processes {
				statusEl := Del(Text(proc.Status))
				if proc.IsRunning {
					statusEl = Ins(Text("Running"))
				}

				health := proc.Health
				if health == "" {
					health = "N/A"
				}

				var actionsEl H = Small(Text("-"))
				if pa, ok := actions[proc.Name]; ok {
					if proc.IsRunning {
						actionsEl = Div(Role("group"),
							Button(Text("Stop"), Class("secondary outline"), pa["stop"]),
							Button(Text("Restart"), Class("contrast outline"), pa["restart"]),
						)
					} else {
						actionsEl = Button(Text("Start"), pa["start"])
					}
				}

				rows = append(rows, Tr(
					Td(Strong(Text(proc.Name))),
					Td(statusEl),
					Td(Code(Textf("%d", proc.Pid))),
					Td(Text(health)),
					Td(Textf("%d", proc.Restarts)),
					Td(actionsEl),
				))
			}

			var messageEl H
			if lastError != "" {
				messageEl = Article(Attr("data-theme", "light"),
					P(Class("pico-color-red"), Strong(Text("Error: ")), Text(lastError)))
			} else if lastAction != "" {
				messageEl = Article(Attr("data-theme", "light"),
					P(Class("pico-color-green"), Strong(Text("Action: ")), Text(lastAction)))
			}

			var tableEl H
			if len(processes) == 0 && lastError != "" {
				tableEl = Article(
					P(Text("Could not connect to process-compose API.")),
					P(Small(Textf("Make sure process-compose is running with API server enabled on %s", pcURL))),
					P(Small(Text("Run: process-compose up --port 8181"))),
				)
			} else {
				tableEl = Figure(Table(Role("grid"),
					THead(Tr(
						Th(Text("Process")), Th(Text("Status")), Th(Text("PID")),
						Th(Text("Health")), Th(Text("Restarts")), Th(Text("Actions")),
					)),
					TBody(rows...),
				))
			}

			return Main(Class("container"),
				navBar("Processes"),
				Section(
					H1(Text("Process Manager")),
					P(Text("Bidirectional control over NATS: process status & control via pc.processes / pc.processes.control")),
					Button(Text("Refresh Now"), refresh.OnClick()),
				),
				messageEl,
				tableEl,
				Section(Hr(), H5(Text("Dog Fooding Demo")),
					P(Text("The system monitors itself - nats polls the same process-compose that runs it:")),
					Ul(
						Li(Strong(Text("nats (hub): ")), Text("Polls process-compose API, publishes state to NATS")),
						Li(Strong(Text("via (web): ")), Text("Subscribes to NATS, displays state, sends control commands")),
						Li(Strong(Text("process-compose: ")), Text("Runs both services, reports their status back to them")),
						Li(Mark(Text("Try it: ")), Text("Click 'Restart' on via - page reconnects automatically!")),
					),
				),
			)
		})
	})
}
