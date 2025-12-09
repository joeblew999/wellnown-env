// Example: Embedding Via for reactive web UIs in pure Go
//
// Via lets you build full-featured web applications entirely in Go:
// - No JavaScript, no templates, no transpilation
// - Real-time reactivity via Server-Sent Events (SSE)
// - Type-safe UI composition with the h package
//
// Run:
//   go run main.go
//
// Change theme via environment variable:
//   VIA_THEME=purple go run main.go
//   VIA_THEME=amber go run main.go
//
// Then open http://localhost:3000 in your browser
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-via/via"
	"github.com/go-via/via-plugin-picocss/picocss"
	. "github.com/go-via/via/h"
)

// themeMap maps theme names to picocss.Theme constants
var themeMap = map[string]picocss.Theme{
	"amber":   picocss.ThemeAmber,
	"blue":    picocss.ThemeBlue,
	"cyan":    picocss.ThemeCyan,
	"fuchsia": picocss.ThemeFuchia,
	"green":   picocss.ThemeGreen,
	"grey":    picocss.ThemeGrey,
	"indigo":  picocss.ThemeIndigo,
	"jade":    picocss.ThemeJade,
	"lime":    picocss.ThemeLime,
	"orange":  picocss.ThemeOrange,
	"pink":    picocss.ThemePink,
	"pumpkin": picocss.ThemePumpkin,
	"purple":  picocss.ThemePurple,
	"red":     picocss.ThemeRed,
	"sand":    picocss.ThemeSand,
	"slate":   picocss.ThemeSlate,
	"violet":  picocss.ThemeViolet,
	"yellow":  picocss.ThemeYellow,
	"zinc":    picocss.ThemeZinc,
}

// getThemeFromEnv reads VIA_THEME env var and returns the corresponding theme
func getThemeFromEnv() (picocss.Theme, string) {
	themeName := strings.ToLower(os.Getenv("VIA_THEME"))
	if themeName == "" {
		return picocss.ThemeIndigo, "indigo"
	}
	if theme, ok := themeMap[themeName]; ok {
		return theme, themeName
	}
	// Default to indigo if invalid theme name
	return picocss.ThemeIndigo, "indigo"
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("Via Embedded Example")
	fmt.Println("====================")
	fmt.Println()

	// Get theme from environment
	theme, themeName := getThemeFromEnv()
	fmt.Printf("Using theme: %s (set VIA_THEME env to change)\n\n", themeName)

	v := via.New()

	v.Config(via.Options{
		ServerAddress: ":3000",
		DocumentTitle: "wellnown-env Dashboard",
		LogLvl:        via.LogLevelInfo,
		Plugins: []via.Plugin{
			picocss.WithOptions(picocss.Options{
				Theme:         theme,
				IncludeColors: true,
			}),
		},
	})

	// Store current theme name for display in UI
	currentTheme := themeName

	// Home page - counter demo
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
				Section(
					Nav(
						Ul(Li(Strong(Text("wellnown-env")))),
						Ul(
							Li(A(Text("Services"), Href("/services"))),
							Li(A(Text("Live Clock"), Href("/live"))),
							Li(A(Text("Themes"), Href("/themes"))),
						),
					),
				),

				Section(
					H1(Text("Dashboard")),
					P(Text("A reactive UI built entirely in Go - no JavaScript!")),
				),

				Article(
					Header(H2(Text("Counter Demo"))),
					P(Textf("Count: %d", count)),
					Div(Role("group"),
						Button(Text("- Decrement"), decrement.OnClick()),
						Button(Text("+ Increment"), increment.OnClick()),
					),
				),
			)
		})
	})

	// Services page - simulates NATS service registry
	v.Page("/services", func(c *via.Context) {
		services := []struct {
			Name   string
			Status string
			Host   string
		}{
			{"api-gateway", "Running", "localhost:8080"},
			{"auth-service", "Running", "localhost:8081"},
			{"user-service", "Running", "localhost:8082"},
			{"billing-api", "Stopped", "localhost:8083"},
		}

		refreshCount := 0

		refresh := c.Action(func() {
			refreshCount++
			c.Sync()
		})

		c.View(func() H {
			rows := []H{}
			for _, svc := range services {
				var statusEl H
				if svc.Status == "Running" {
					statusEl = Ins(Text(svc.Status))
				} else {
					statusEl = Del(Text(svc.Status))
				}
				rows = append(rows, Tr(
					Td(Text(svc.Name)),
					Td(statusEl),
					Td(Code(Text(svc.Host))),
				))
			}

			return Main(Class("container"),
				Section(
					Nav(
						Ul(Li(A(Text("Home"), Href("/")))),
					),
				),

				Section(
					H1(Text("Registered Services")),
					P(Textf("Refresh count: %d", refreshCount)),
					Button(Text("Refresh"), refresh.OnClick()),
				),

				Figure(
					Table(Role("grid"),
						THead(
							Tr(
								Th(Text("Service")),
								Th(Text("Status")),
								Th(Text("Host")),
							),
						),
						TBody(rows...),
					),
				),
			)
		})
	})

	// Live clock - real-time updates via SSE
	v.Page("/live", func(c *via.Context) {
		currentTime := time.Now().Format("15:04:05")

		timer := c.OnInterval(1*time.Second, func() {
			currentTime = time.Now().Format("15:04:05")
			c.Sync()
		})
		timer.Start()

		c.View(func() H {
			return Main(Class("container"),
				Section(
					Nav(
						Ul(Li(A(Text("Home"), Href("/")))),
					),
				),

				Section(
					H1(Text("Live Clock")),
					P(Text("Updates every second via SSE - no JavaScript polling!")),
				),

				Article(
					H2(Text(currentTime)),
				),
			)
		})
	})

	// Themes page - showcases PicoCSS color themes
	v.Page("/themes", func(c *via.Context) {
		// Helper to create theme list item with active marker
		themeItem := func(name, cssClass string) H {
			if strings.ToLower(name) == currentTheme {
				return Li(Mark(Class(cssClass), Text(name)), Text(" *"))
			}
			return Li(Mark(Class(cssClass), Text(name)))
		}

		c.View(func() H {
			return Main(Class("container"),
				Section(
					Nav(
						Ul(Li(A(Text("Home"), Href("/")))),
					),
				),

				Section(
					H1(Text("PicoCSS Theme Colors")),
					P(Textf("Current theme: %s", strings.ToUpper(currentTheme[:1])+currentTheme[1:])),
					P(Small(Text("Change theme: VIA_THEME=purple go run main.go"))),
				),

				Section(
					H5(Text("Available Themes")),
					Div(Class("grid"),
						Ul(
							themeItem("Amber", "pico-background-amber"),
							themeItem("Blue", "pico-background-blue"),
							themeItem("Cyan", "pico-background-cyan"),
							themeItem("Fuchsia", "pico-background-fuchsia"),
							themeItem("Green", "pico-background-green"),
						),
						Ul(
							themeItem("Grey", "pico-background-grey"),
							themeItem("Indigo", "pico-background-indigo"),
							themeItem("Jade", "pico-background-jade"),
							themeItem("Lime", "pico-background-lime"),
							themeItem("Orange", "pico-background-orange"),
						),
						Ul(
							themeItem("Pink", "pico-background-pink"),
							themeItem("Pumpkin", "pico-background-pumpkin"),
							themeItem("Purple", "pico-background-purple"),
							themeItem("Red", "pico-background-red"),
							themeItem("Sand", "pico-background-sand"),
						),
						Ul(
							themeItem("Slate", "pico-background-slate"),
							themeItem("Violet", "pico-background-violet"),
							themeItem("Yellow", "pico-background-yellow"),
							themeItem("Zinc", "pico-background-zinc"),
						),
					),
					P(Small(Text("(*) Current theme"))),
				),

				Section(
					Hr(),
					H5(Text("Color Utility Classes")),
					P(Text("With IncludeColors: true, you get:")),
					Ul(
						Li(Code(Text("pico-background-{color}")), Text(" - Background colors")),
						Li(Code(Text("pico-color-{color}")), Text(" - Text colors")),
					),
				),
			)
		})
	})

	fmt.Println("Starting Via server on http://localhost:3000")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	v.Start()

	return nil
}
