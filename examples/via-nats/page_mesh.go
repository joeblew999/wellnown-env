package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// registerMeshPage registers the mesh control page
func registerMeshPage(v *via.V) {
	v.Page("/mesh", func(c *via.Context) {
		var lastOutput string

		// Actions for mesh control
		startMesh := c.Action(func() {
			result := RunTask("mesh")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicMesh)
			c.Sync()
		})

		stopMesh := c.Action(func() {
			result := RunTask("mesh:down")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicMesh)
			c.Sync()
		})

		listMesh := c.Action(func() {
			result := RunTask("mesh:list")
			lastOutput = result.Output + result.Error
			c.Sync()
		})

		cleanData := c.Action(func() {
			result := RunTask("clean")
			lastOutput = result.Output + result.Error
			c.Sync()
		})

		cleanLogs := c.Action(func() {
			result := RunTask("clean:logs")
			lastOutput = result.Output + result.Error
			c.Sync()
		})

		cleanAll := c.Action(func() {
			result := RunTask("clean:all")
			lastOutput = result.Output + result.Error
			c.Sync()
		})

		// Subscribe to mesh updates
		broadcast.Subscribe(TopicMesh, func() { c.Sync() })

		c.View(func() H {
			lastResult := GetLastResult()

			return Main(Class("container"),
				navBar("Mesh"),

				Section(
					H1(Text("Mesh Control")),
					P(Text("Start, stop, and monitor the NATS mesh (1 hub + 4 leaf nodes)")),
				),

				resultMessage(lastResult),

				Article(
					Header(H2(Text("Mesh Operations"))),
					Div(Role("group"),
						Button(Text("Start Mesh"), Class(""), startMesh.OnClick()),
						Button(Text("Stop Mesh"), Class("secondary"), stopMesh.OnClick()),
						Button(Text("List Processes"), Class("outline"), listMesh.OnClick()),
					),
				),

				Article(
					Header(H2(Text("Cleanup"))),
					P(Text("Remove data directories and logs")),
					Div(Role("group"),
						Button(Text("Clean Data"), Class("secondary"), cleanData.OnClick()),
						Button(Text("Clean Logs"), Class("secondary"), cleanLogs.OnClick()),
						Button(Text("Clean All"), Class("contrast"), cleanAll.OnClick()),
					),
					P(Small(Text("Note: Stop mesh before cleaning"))),
				),

				Article(
					Header(H2(Text("Mesh Architecture"))),
					Table(Role("grid"),
						THead(
							Tr(
								Th(Text("Process")),
								Th(Text("Port")),
								Th(Text("Role")),
							),
						),
						TBody(
							Tr(
								Td(Strong(Text("hub"))),
								Td(Text("4222")),
								Td(Text("Central NATS server")),
							),
							Tr(
								Td(Text("svc-a")),
								Td(Text("4223")),
								Td(Text("Leaf node")),
							),
							Tr(
								Td(Text("svc-b")),
								Td(Text("4224")),
								Td(Text("Leaf node")),
							),
							Tr(
								Td(Text("svc-c")),
								Td(Text("4225")),
								Td(Text("Leaf node")),
							),
							Tr(
								Td(Text("svc-d")),
								Td(Text("4226")),
								Td(Text("Leaf node")),
							),
						),
					),
				),

				outputPanel("Command Output", lastOutput),
			)
		})
	})
}
