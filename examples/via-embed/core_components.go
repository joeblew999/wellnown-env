package main

import (
	. "github.com/go-via/via/h"
)

// natsStatusElement returns a styled element showing NATS connection status
func natsStatusElement() H {
	if isNatsConnected() {
		return Ins(Text("Connected"))
	}
	return Del(Text("Disconnected"))
}

// messageElement returns an article element for displaying action/error messages
func messageElement(lastError, lastAction string) H {
	if lastError != "" {
		return Article(Attr("data-theme", "light"),
			P(Class("pico-color-red"), Strong(Text("Error: ")), Text(lastError)),
		)
	}
	if lastAction != "" {
		return Article(Attr("data-theme", "light"),
			P(Class("pico-color-green"), Text(lastAction)),
		)
	}
	return nil
}

// navBar creates the navigation bar component
func navBar(active string) H {
	navItem := func(name, href string) H {
		if name == active {
			return Li(A(Strong(Text(name)), Href(href)))
		}
		return Li(A(Text(name), Href(href)))
	}
	return Section(
		Nav(
			Ul(Li(Strong(Text("wellnown-env")))),
			Ul(
				navItem("Home", "/"),
				navItem("Counter", "/counter"),
				navItem("Monitor", "/monitor"),
				navItem("Config", "/config"),
				navItem("Processes", "/processes"),
				navItem("Services", "/services"),
				navItem("Chat", "/chat"),
				navItem("Themes", "/themes"),
				navItem("Version", "/version"),
				navItem("RTL", "/rtl"),
				navItem("Live", "/live"),
			),
		),
	)
}
