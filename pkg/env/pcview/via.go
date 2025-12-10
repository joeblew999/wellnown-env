package pcview

import (
	"fmt"
	"time"

	"github.com/go-via/via"
	. "github.com/go-via/via/h"
	"github.com/joeblew999/wellnown-env/pkg/env"
)

// ProcessController is the interface for controlling processes
// This allows both HTTP client and embedded runner to be used
type ProcessController interface {
	GetProcesses() ([]ProcessState, error)
	Control(action, name string) error
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
}

// PageOptions configures the Via page
type PageOptions struct {
	// NavBar returns the navigation bar H element
	NavBar func(title string) H
	// OnSync is called when the page needs to broadcast updates
	OnSync func()
	// Controllable lists processes to show control buttons for.
	// If empty or nil, ALL processes are controllable (default).
	// Use explicit list to restrict control to specific processes.
	Controllable []string
	// PCPort is the process-compose API port for error messages (default: from env)
	PCPort string
}

// RegisterPage registers the /processes page with Via
func RegisterPage(v *via.V, client ProcessController, state *State, opts PageOptions) {
	// If Controllable is empty, all processes are controllable
	allControllable := len(opts.Controllable) == 0
	controllable := make(map[string]bool)
	for _, name := range opts.Controllable {
		controllable[name] = true
	}

	// Get PC port from options or environment
	pcPort := opts.PCPort
	if pcPort == "" {
		pcPort = env.GetEnv("PC_PORT", env.DefaultPCPort)
	}

	v.Page("/processes", func(c *via.Context) {
		var lastAction string
		var lastError string

		// Helper to check if a process is controllable
		isControllable := func(name string) bool {
			return allControllable || controllable[name]
		}

		// Helper to create control actions
		makeControl := func(action, name, msg string) H {
			return c.Action(func() {
				if err := client.Control(action, name); err != nil {
					lastError = err.Error()
					lastAction = ""
				} else {
					lastAction = msg
					lastError = ""
				}
				c.Sync()
			}).OnClick()
		}

		// Helper to create restart action (special handling for "via" process)
		makeRestart := func(name string) H {
			if name == "via" {
				return c.Action(func() {
					lastAction = "Restarting via... (page will reconnect)"
					lastError = ""
					c.Sync()
					time.Sleep(100 * time.Millisecond)
					if err := client.Restart(name); err != nil {
						lastError = err.Error()
						lastAction = ""
						c.Sync()
					}
				}).OnClick()
			}
			return makeControl("restart", name, "Restarted "+name)
		}

		// Refresh action
		refresh := c.Action(func() {
			procs, err := client.GetProcesses()
			if err != nil {
				lastError = err.Error()
			} else {
				state.SetProcesses(procs, "")
				lastError = ""
			}
			c.Sync()
		})

		c.View(func() H {
			processes, stateErr := state.GetProcesses()
			if stateErr != "" && lastError == "" {
				lastError = stateErr
			}

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
				if isControllable(proc.Name) {
					if proc.IsRunning {
						actionsEl = Div(Role("group"),
							Button(Text("Stop"), Class("secondary outline"), makeControl("stop", proc.Name, "Stopped "+proc.Name)),
							Button(Text("Restart"), Class("contrast outline"), makeRestart(proc.Name)),
						)
					} else {
						actionsEl = Button(Text("Start"), makeControl("start", proc.Name, "Started "+proc.Name))
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
					P(Small(Text("Make sure process-compose is running with API server enabled."))),
					P(Small(Text(fmt.Sprintf("Run: process-compose up --port %s", pcPort)))),
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

			var navEl H
			if opts.NavBar != nil {
				navEl = opts.NavBar("Processes")
			}

			return Main(Class("container"),
				navEl,
				Section(
					H1(Text("Process Manager")),
					P(Text("View and control process-compose processes")),
					Button(Text("Refresh"), refresh.OnClick()),
				),
				messageEl,
				tableEl,
			)
		})
	})
}
