package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// registerHomePage registers the home page handler
func registerHomePage(v *via.V) {
	v.Page("/", func(c *via.Context) {
		count := 0

		increment := c.Action(func() {
			count++
			c.Sync()
		})

		decrement := c.Action(func() {
			count--
			c.Sync()
		})

		c.View(func() H {
			return Main(Class("container"),
				navBar("Home"),

				Section(
					H1(Text("Dashboard")),
					P(Text("A reactive UI built entirely in Go - no JavaScript!")),
					P(Text("NATS Status: "), natsStatusElement()),
				),

				Article(
					Header(H2(Text("Counter Demo"))),
					P(Textf("Count: %d", count)),
					Div(Role("group"),
						Button(Text("- Decrement"), decrement.OnClick()),
						Button(Text("+ Increment"), increment.OnClick()),
					),
				),

				Section(
					H3(Text("Funky Features")),
					Ul(
						Li(A(Text("/counter"), Href("/counter")), Text(" - Distributed counter synced via NATS KV")),
						Li(A(Text("/monitor"), Href("/monitor")), Text(" - Real-time NATS message viewer")),
						Li(A(Text("/config"), Href("/config")), Text(" - Config hot-reload demo via NATS KV")),
						Li(A(Text("/services"), Href("/services")), Text(" - Live service registry")),
						Li(A(Text("/chat"), Href("/chat")), Text(" - Cross-instance chat via NATS pub/sub")),
						Li(A(Text("/themes"), Href("/themes")), Text(" - Click themes to sync via NATS KV")),
					),
				),
			)
		})
	})
}
