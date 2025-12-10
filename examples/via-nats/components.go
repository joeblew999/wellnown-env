package main

import (
	. "github.com/go-via/via/h"
)

// authModeLabel returns a styled label for the auth mode
func authModeLabel(mode string) H {
	var class string
	switch mode {
	case "none":
		class = "pico-color-grey"
	case "token":
		class = "pico-color-orange"
	case "nkey":
		class = "pico-color-cyan"
	case "jwt":
		class = "pico-color-green"
	default:
		class = "pico-color-grey"
	}
	return Strong(Class(class), Text(mode))
}

// outputPanel displays command output in a preformatted box
func outputPanel(title, content string) H {
	if content == "" {
		return nil
	}
	return Article(
		Header(H4(Text(title))),
		Pre(Code(Text(content))),
	)
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
			Ul(Li(Strong(Text("NATS Auth")))),
			Ul(
				navItem("Dashboard", "/"),
				navItem("Auth", "/auth"),
				navItem("Mesh", "/mesh"),
				navItem("Tests", "/tests"),
			),
		),
	)
}

// statusBadge returns a styled status indicator
func statusBadge(ok bool, label string) H {
	if ok {
		return Ins(Text(label))
	}
	return Del(Text(label))
}

// resultMessage shows the result of the last operation
func resultMessage(result TaskResult) H {
	if result.Command == "" {
		return nil
	}
	if result.ExitCode != 0 {
		return Article(Attr("data-theme", "light"),
			P(Class("pico-color-red"),
				Strong(Text("Error: ")),
				Text(result.Error),
			),
		)
	}
	return Article(Attr("data-theme", "light"),
		P(Class("pico-color-green"),
			Strong(Textf("Success: %s", result.Command)),
		),
	)
}
