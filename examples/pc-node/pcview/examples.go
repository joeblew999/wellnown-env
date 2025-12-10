package pcview

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// ExampleProcesses defines the built-in demo processes for regression testing
var ExampleProcesses = []string{"ticker", "counter", "logger"}

// ExamplesPageOptions configures the examples page
type ExamplesPageOptions struct {
	// NavBar returns the navigation bar H element
	NavBar func(title string) H
}

// RegisterExamplesPage registers the /examples page for demo process testing
func RegisterExamplesPage(v *via.V, client ProcessController, state *State, opts ExamplesPageOptions) {
	v.Page("/examples", func(c *via.Context) {
		var lastAction string
		var lastError string

		// Helper to create control actions for examples
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

		// Quick actions for all demo processes
		startAll := c.Action(func() {
			for _, name := range ExampleProcesses {
				if err := client.Start(name); err != nil {
					lastError = err.Error()
					return
				}
			}
			lastAction = "Started all demo processes"
			lastError = ""
			c.Sync()
		})

		stopAll := c.Action(func() {
			for _, name := range ExampleProcesses {
				if err := client.Stop(name); err != nil {
					// Ignore errors for already stopped processes
					continue
				}
			}
			lastAction = "Stopped all demo processes"
			lastError = ""
			c.Sync()
		})

		restartAll := c.Action(func() {
			for _, name := range ExampleProcesses {
				if err := client.Restart(name); err != nil {
					lastError = err.Error()
					return
				}
			}
			lastAction = "Restarted all demo processes"
			lastError = ""
			c.Sync()
		})

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

			// Filter to only example processes
			exampleSet := make(map[string]bool)
			for _, name := range ExampleProcesses {
				exampleSet[name] = true
			}

			var rows []H
			for _, proc := range processes {
				if !exampleSet[proc.Name] {
					continue
				}

				statusEl := Del(Text(proc.Status))
				if proc.IsRunning {
					statusEl = Ins(Text("Running"))
				}

				health := proc.Health
				if health == "" {
					health = "N/A"
				}

				var actionsEl H
				if proc.IsRunning {
					actionsEl = Div(Role("group"),
						Button(Text("Stop"), Class("secondary outline"), makeControl("stop", proc.Name, "Stopped "+proc.Name)),
						Button(Text("Restart"), Class("contrast outline"), makeControl("restart", proc.Name, "Restarted "+proc.Name)),
					)
				} else {
					actionsEl = Button(Text("Start"), makeControl("start", proc.Name, "Started "+proc.Name))
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

			var navEl H
			if opts.NavBar != nil {
				navEl = opts.NavBar("Examples")
			}

			return Main(Class("container"),
				navEl,
				Section(
					H1(Text("Demo Processes")),
					P(Text("Built-in example processes for regression testing")),
					Div(Role("group"),
						Button(Text("Refresh"), refresh.OnClick()),
						Button(Text("Start All"), Class("secondary"), startAll.OnClick()),
						Button(Text("Stop All"), Class("secondary outline"), stopAll.OnClick()),
						Button(Text("Restart All"), Class("contrast outline"), restartAll.OnClick()),
					),
				),
				messageEl,
				Article(
					H4(Text("Example Processes")),
					P(Small(Text("These 3 demo processes are defined in pc.yaml:"))),
					Ul(
						Li(Strong(Text("ticker")), Text(" - Prints timestamp every 5 seconds")),
						Li(Strong(Text("counter")), Text(" - Increments count every 3 seconds (depends on ticker)")),
						Li(Strong(Text("logger")), Text(" - Logs status every 10 seconds (depends on ticker & counter)")),
					),
				),
				Figure(Table(Role("grid"),
					THead(Tr(
						Th(Text("Process")), Th(Text("Status")), Th(Text("PID")),
						Th(Text("Health")), Th(Text("Restarts")), Th(Text("Actions")),
					)),
					TBody(rows...),
				)),
				Hr(),
				Article(
					H5(Text("Regression Test Scenarios")),
					Ol(
						Li(Text("Click 'Stop All' - all processes should stop")),
						Li(Text("Click 'Start All' - processes start in dependency order")),
						Li(Text("Stop 'ticker' - counter and logger lose their dependency")),
						Li(Text("Click 'Restart All' - verifies restart functionality")),
					),
				),
			)
		})
	})
}
