package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// registerAuthPage registers the auth management page
func registerAuthPage(v *via.V) {
	v.Page("/auth", func(c *via.Context) {
		var lastOutput string

		// Actions for each auth mode
		setNone := c.Action(func() {
			result := RunTask("auth:clean")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicAuth)
			c.Sync()
		})

		setToken := c.Action(func() {
			result := RunTask("auth:token")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicAuth)
			c.Sync()
		})

		setNKey := c.Action(func() {
			result := RunTask("auth:nkey")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicAuth)
			c.Sync()
		})

		setJWT := c.Action(func() {
			result := RunTask("auth:jwt")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicAuth)
			c.Sync()
		})

		// Fresh start actions (clean + set mode)
		freshNone := c.Action(func() {
			RunTask("clean")
			result := RunTask("auth:clean")
			lastOutput = "Cleaned data and reset to dev mode\n" + result.Output + result.Error
			broadcast.Notify(TopicAuth)
			broadcast.Notify(TopicMesh)
			c.Sync()
		})

		freshToken := c.Action(func() {
			RunTask("clean")
			RunTask("auth:clean")
			result := RunTask("auth:token")
			lastOutput = "Cleaned data and set token auth\n" + result.Output + result.Error
			broadcast.Notify(TopicAuth)
			broadcast.Notify(TopicMesh)
			c.Sync()
		})

		freshNKey := c.Action(func() {
			RunTask("clean")
			RunTask("auth:clean")
			result := RunTask("auth:nkey")
			lastOutput = "Cleaned data and set NKey auth\n" + result.Output + result.Error
			broadcast.Notify(TopicAuth)
			broadcast.Notify(TopicMesh)
			c.Sync()
		})

		freshJWT := c.Action(func() {
			RunTask("clean")
			RunTask("auth:clean")
			result := RunTask("auth:jwt")
			lastOutput = "Cleaned data and set JWT auth\n" + result.Output + result.Error
			broadcast.Notify(TopicAuth)
			broadcast.Notify(TopicMesh)
			c.Sync()
		})

		// Subscribe to auth updates
		broadcast.Subscribe(TopicAuth, func() { c.Sync() })

		c.View(func() H {
			authMode := GetAuthStatus()
			lastResult := GetLastResult()

			// Get mode-specific details
			var modeDetails H
			switch authMode {
			case "token":
				token := GetAuthToken()
				if token != "" {
					masked := token[:8] + "..." + token[len(token)-4:]
					modeDetails = Article(
						H4(Text("Token Details")),
						P(Text("Token: "), Code(Text(masked))),
						P(Small(Text("Full token in .auth/token"))),
					)
				}
			case "nkey":
				pubKey := GetNKeyPub()
				if pubKey != "" {
					modeDetails = Article(
						H4(Text("NKey Details")),
						P(Text("Public Key: "), Code(Text(pubKey))),
						P(Small(Text("Seed in .auth/user.nk"))),
					)
				}
			case "jwt":
				modeDetails = Article(
					H4(Text("JWT Details")),
					P(Text("Credentials: "), Code(Text(".auth/creds/user.creds"))),
					P(Small(Text("NSC operator: wellnown, account: APP"))),
				)
			}

			return Main(Class("container"),
				navBar("Auth"),

				Section(
					H1(Text("Auth Lifecycle Management")),
					P(Text("Current mode: "), authModeLabel(authMode)),
				),

				resultMessage(lastResult),

				modeDetails,

				Article(
					Header(H2(Text("Switch Auth Mode"))),

					Div(Class("grid"),
						Article(
							Header(H4(authModeLabel("none"))),
							P(Text("Development mode - no authentication required")),
							Button(
								Text("Reset to None"),
								setNone.OnClick(),
								func() H {
									if authMode == "none" {
										return Attr("disabled", "disabled")
									}
									return nil
								}(),
							),
						),

						Article(
							Header(H4(authModeLabel("token"))),
							P(Text("Test/CI mode - shared token authentication")),
							Button(
								Text("Set Token Auth"),
								setToken.OnClick(),
								func() H {
									if authMode == "token" {
										return Attr("disabled", "disabled")
									}
									return nil
								}(),
							),
						),
					),

					Div(Class("grid"),
						Article(
							Header(H4(authModeLabel("nkey"))),
							P(Text("Staging mode - NKey keypair authentication")),
							Button(
								Text("Set NKey Auth"),
								setNKey.OnClick(),
								func() H {
									if authMode == "nkey" {
										return Attr("disabled", "disabled")
									}
									return nil
								}(),
							),
						),

						Article(
							Header(H4(authModeLabel("jwt"))),
							P(Text("Production mode - JWT/NSC account authentication")),
							Button(
								Text("Set JWT Auth"),
								setJWT.OnClick(),
								func() H {
									if authMode == "jwt" {
										return Attr("disabled", "disabled")
									}
									return nil
								}(),
							),
						),
					),
				),

				Article(
					Header(H2(Text("Fresh Start (Clean + Set Mode)"))),
					P(Text("Cleans all data before switching mode. Use for testing lifecycle transitions.")),
					Div(Role("group"),
						Button(Text("Fresh None"), Class("secondary"), freshNone.OnClick()),
						Button(Text("Fresh Token"), Class("secondary"), freshToken.OnClick()),
						Button(Text("Fresh NKey"), Class("secondary"), freshNKey.OnClick()),
						Button(Text("Fresh JWT"), Class("secondary"), freshJWT.OnClick()),
					),
					P(Small(Text("Warning: Stops mesh and removes all JetStream data"))),
				),

				outputPanel("Command Output", lastOutput),
			)
		})
	})
}
