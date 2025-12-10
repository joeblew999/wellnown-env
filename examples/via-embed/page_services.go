package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// registerServicesPage registers the NATS service registry page handler
func registerServicesPage(v *via.V) {
	v.Page("/services", func(c *via.Context) {
		var lastError string

		// Initial fetch
		if svcs, err := getServicesFromNATS(); err != nil {
			lastError = err.Error()
		} else {
			servicesMu.Lock()
			liveServices = svcs
			servicesMu.Unlock()
		}

		// Subscribe to services registry updates via NATS broadcast (no polling!)
		broadcast.Subscribe(TopicNats, func() {
			c.Sync()
		})

		c.View(func() H {
			// Get live services from shared state
			servicesMu.RLock()
			services := make([]ServiceRegistration, len(liveServices))
			copy(services, liveServices)
			servicesMu.RUnlock()

			var content H
			if lastError != "" {
				content = Article(
					P(Class("pico-color-red"), Text(lastError)),
					P(Small(Text("Make sure NATS is running on port 4222"))),
				)
			} else if len(services) == 0 {
				content = Article(
					P(Text("No services registered yet.")),
					P(Small(Text("Start the nats-node example to see services appear"))),
				)
			} else {
				var rows []H
				for _, svc := range services {
					rows = append(rows, Tr(
						Td(Strong(Text(svc.Name))),
						Td(Code(Text(svc.Host))),
						Td(Small(Text(svc.Time))),
					))
				}
				content = Figure(
					Table(Role("grid"),
						THead(Tr(
							Th(Text("Service")),
							Th(Text("Host")),
							Th(Text("Registered")),
						)),
						TBody(rows...),
					),
				)
			}

			return Main(Class("container"),
				navBar("Services"),

				Section(
					H1(Text("Service Registry")),
					P(Text("Live view of services registered in NATS KV")),
					P(Text("Status: "), natsStatusElement(), Small(Text(" (live updates via NATS watch)"))),
				),

				content,

				Section(
					Hr(),
					H5(Text("How It Works")),
					Ul(
						Li(Text("Services register themselves to NATS KV bucket 'services_registry'")),
						Li(Text("This page watches NATS KV for changes - no polling!")),
						Li(Text("Services with TTL automatically expire if they stop heartbeating")),
					),
				),
			)
		})
	})
}
