package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

func registerThemesPage(v *via.V) {
	v.Page("/themes", func(c *via.Context) {
		// Create actions for each theme dynamically
		actions := make(map[string]H)
		for _, t := range themes {
			name := t // capture for closure
			actions[name] = c.Action(func() { setTheme(name); c.Sync() }).OnClick()
		}

		broadcast.Subscribe(TopicTheme, func() { c.Sync() })

		c.View(func() H {
			current := getTheme()

			// Build theme buttons dynamically
			buttons := make([]H, 0, len(themes))
			for _, t := range themes {
				class := "pico-background-" + t
				label := t
				if t == current {
					label = t + " *"
				}
				buttons = append(buttons, Button(Text(label), Class(class), actions[t]))
			}

			return Main(Class("container"),
				navBar("Themes"),
				Section(
					H1(Text("Theme Sync")),
					P(Text("Click a theme - syncs to all tabs via NATS KV")),
					P(Text("Current: "), Strong(Text(func() string {
						if current == "" {
							return "(none)"
						}
						return current
					}()))),
				),
				Article(
					Div(append([]H{Class("grid")}, buttons...)...),
				),
			)
		})
	})
}
