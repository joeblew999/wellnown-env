package main

import (
	"sort"

	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// sortedConfigKeys returns sorted keys from configToggle map
func sortedConfigKeys() []string {
	keys := make([]string, 0, len(configToggle))
	for k := range configToggle {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func registerConfigPage(v *via.V) {
	v.Page("/config", func(c *via.Context) {
		keys := sortedConfigKeys()

		// Create toggle actions for each key
		toggleActions := make(map[string]H)
		deleteActions := make(map[string]H)
		for _, key := range keys {
			k := key // capture for closure
			toggleActions[k] = c.Action(func() { toggleConfig(k); c.Sync() }).OnClick()
			deleteActions[k] = c.Action(func() { deleteConfig(k); c.Sync() }).OnClick()
		}

		broadcast.Subscribe(TopicConfig, func() { c.Sync() })

		c.View(func() H {
			// Build table rows dynamically
			rows := make([]H, 0, len(keys))
			for _, key := range keys {
				k := key
				rows = append(rows, Tr(
					Td(Code(Text(k))),
					Td(func() H {
						if v := getConfig(k); v != "" {
							return Text(v)
						}
						return Em(Text("(not set)"))
					}()),
					Td(Div(Role("group"),
						Button(Text("Toggle"), Class("outline"), toggleActions[k]),
						Button(Text("Delete"), Class("outline secondary"), deleteActions[k]),
					)),
				))
			}

			return Main(Class("container"),
				navBar("Config"),
				Section(
					H1(Text("Config Hot-Reload")),
					P(Text("Values stored in NATS KV - changes sync to all tabs")),
				),
				Article(
					Table(Role("grid"),
						THead(Tr(
							Th(Text("Key")),
							Th(Text("Value")),
							Th(Text("Actions")),
						)),
						TBody(rows...),
					),
				),
			)
		})
	})
}
