package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// registerLivePage registers the live clock page handler
func registerLivePage(v *via.V) {
	v.Page("/live", func(c *via.Context) {
		// Subscribe to time updates via broadcast (no polling in page!)
		broadcast.Subscribe(TopicTime, func() {
			c.Sync()
		})

		c.View(func() H {
			// Get live time from shared state
			timeMu.RLock()
			timeStr := currentTime
			timeMu.RUnlock()

			if timeStr == "" {
				timeStr = "Loading..."
			}

			return Main(Class("container"),
				navBar(""),

				Section(
					H1(Text("Live Clock")),
					P(Text("Updates every second via background ticker + broadcast - no per-page polling!")),
				),

				Article(
					H2(Text(timeStr)),
				),
			)
		})
	})
}

// registerHealthzPage registers the health check endpoint
func registerHealthzPage(v *via.V) {
	v.Page("/healthz", func(c *via.Context) {
		c.View(func() H {
			return Text("ok")
		})
	})
}
