package main

import (
	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// registerTestsPage registers the tests page
func registerTestsPage(v *via.V) {
	v.Page("/tests", func(c *via.Context) {
		var lastOutput string

		// Actions for each test
		testAccount := c.Action(func() {
			result := RunTask("test:account")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicTests)
			c.Sync()
		})

		testKV := c.Action(func() {
			result := RunTask("test:kv")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicTests)
			c.Sync()
		})

		testServices := c.Action(func() {
			result := RunTask("test:services")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicTests)
			c.Sync()
		})

		testPubSub := c.Action(func() {
			result := RunTask("test:pubsub")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicTests)
			c.Sync()
		})

		testAll := c.Action(func() {
			result := RunTask("test")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicTests)
			c.Sync()
		})

		testLifecycle := c.Action(func() {
			result := RunTask("test:lifecycle")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicTests)
			broadcast.Notify(TopicAuth)
			c.Sync()
		})

		testTransitions := c.Action(func() {
			result := RunTask("test:transitions")
			lastOutput = result.Output + result.Error
			broadcast.Notify(TopicTests)
			broadcast.Notify(TopicAuth)
			c.Sync()
		})

		// Subscribe to test updates
		broadcast.Subscribe(TopicTests, func() { c.Sync() })

		c.View(func() H {
			authMode := GetAuthStatus()
			lastResult := GetLastResult()

			return Main(Class("container"),
				navBar("Tests"),

				Section(
					H1(Text("NATS Tests")),
					P(Text("Run tests to verify NATS connectivity and operations")),
					P(Text("Current auth mode: "), authModeLabel(authMode)),
				),

				resultMessage(lastResult),

				Article(
					Header(H2(Text("Individual Tests"))),
					Div(Class("grid"),
						Article(
							H4(Text("Account Info")),
							P(Small(Text("Show NATS account information"))),
							Button(Text("Run"), testAccount.OnClick()),
						),
						Article(
							H4(Text("KV Buckets")),
							P(Small(Text("List JetStream KV buckets"))),
							Button(Text("Run"), testKV.OnClick()),
						),
					),
					Div(Class("grid"),
						Article(
							H4(Text("Services Registry")),
							P(Small(Text("List registered services"))),
							Button(Text("Run"), testServices.OnClick()),
						),
						Article(
							H4(Text("Pub/Sub Test")),
							P(Small(Text("Send a test message"))),
							Button(Text("Run"), testPubSub.OnClick()),
						),
					),
				),

				Article(
					Header(H2(Text("Run All Tests"))),
					P(Text("Execute all tests in sequence")),
					Button(Text("Run All Tests"), Class(""), testAll.OnClick()),
				),

				Article(
					Header(H2(Text("Regression Tests"))),
					P(Text("Auth lifecycle regression tests (Go tests). These take several minutes.")),
					Div(Role("group"),
						Button(Text("Test Lifecycle"), Class("secondary"), testLifecycle.OnClick()),
						Button(Text("Test Transitions"), Class("secondary"), testTransitions.OnClick()),
					),
					P(Small(Text("Warning: These tests will cycle through all auth modes and restart the hub"))),
				),

				outputPanel("Test Output", lastOutput),
			)
		})
	})
}
