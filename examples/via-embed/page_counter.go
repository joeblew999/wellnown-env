package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

func registerCounterPage(v *via.V) {
	v.Page("/counter", func(c *via.Context) {
		dec := c.Action(func() { setCounter(getCounter() - 1); c.Sync() })
		inc := c.Action(func() { setCounter(getCounter() + 1); c.Sync() })
		add10 := c.Action(func() { setCounter(getCounter() + 10); c.Sync() })
		reset := c.Action(func() { setCounter(0); c.Sync() })

		broadcast.Subscribe(TopicCounter, func() { c.Sync() })

		c.View(func() H {
			return Main(Class("container"),
				navBar("Counter"),
				Section(
					H1(Text("Distributed Counter")),
					P(Text("Stored in NATS KV - all tabs see the same value")),
				),
				Article(
					Header(H2(Textf("Counter: %d", getCounter()))),
					Div(Role("group"),
						Button(Text("-1"), Class("secondary"), dec.OnClick()),
						Button(Text("+1"), inc.OnClick()),
						Button(Text("+10"), Class("contrast"), add10.OnClick()),
						Button(Text("Reset"), Class("outline"), reset.OnClick()),
					),
				),
			)
		})
	})
}
