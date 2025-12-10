package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// registerDashboardPage registers the dashboard/home page
func registerDashboardPage(v *via.V) {
	v.Page("/", func(c *via.Context) {
		// Get current state
		authMode := GetAuthStatus()

		// Action to refresh state
		refresh := c.Action(func() {
			c.Sync()
		})

		c.View(func() H {
			// Re-read on each render
			authMode = GetAuthStatus()
			lastResult := GetLastResult()

			return Main(Class("container"),
				navBar("Dashboard"),

				Section(
					H1(Text("NATS Auth Lifecycle Dashboard")),
					P(Text("Web-based management for NATS authentication modes")),
				),

				Article(
					Header(H2(Text("Current Status"))),
					Div(Role("group"),
						Article(
							H4(Text("Auth Mode")),
							P(Text("Current: "), authModeLabel(authMode)),
						),
					),
					Div(
						Button(Text("Refresh"), refresh.OnClick()),
					),
				),

				resultMessage(lastResult),

				Section(
					H3(Text("Quick Navigation")),
					Ul(
						Li(A(Strong(Text("/auth")), Href("/auth")), Text(" - Switch auth modes (none, token, nkey, jwt)")),
						Li(A(Strong(Text("/mesh")), Href("/mesh")), Text(" - Start/stop the NATS mesh")),
						Li(A(Strong(Text("/tests")), Href("/tests")), Text(" - Run NATS tests (account, kv, services)")),
					),
				),

				Section(
					H3(Text("Auth Lifecycle")),
					Table(Role("grid"),
						THead(
							Tr(
								Th(Text("Mode")),
								Th(Text("Environment")),
								Th(Text("Description")),
							),
						),
						TBody(
							Tr(
								Td(authModeLabel("none")),
								Td(Text("Development")),
								Td(Text("No authentication - fast iteration")),
							),
							Tr(
								Td(authModeLabel("token")),
								Td(Text("Test/CI")),
								Td(Text("Shared token auth")),
							),
							Tr(
								Td(authModeLabel("nkey")),
								Td(Text("Staging")),
								Td(Text("NKey public/private keypairs")),
							),
							Tr(
								Td(authModeLabel("jwt")),
								Td(Text("Production")),
								Td(Text("Full NSC accounts with revocation")),
							),
						),
					),
				),
			)
		})
	})
}
