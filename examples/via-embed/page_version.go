package main

import (
	"fmt"

	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// Option groups for CSS settings
var (
	cssVariants = []struct{ value, label, desc string }{
		{"regular", "Regular", "Full-featured with utility classes. Conditional styling on every HTML element."},
		{"classless", "Class-less", "Semantic HTML approach. Perfect for simple pages, documentation, or when you want zero classes."},
	}
	viewportModes = []struct{ value, label, desc string }{
		{"responsive", "Responsive", "Container-based breakpoints. Content width adapts to screen size."},
		{"centered", "Centered", "Content centered with max-width. Great for readability."},
		{"fluid", "Fluid", "Full-width content that stretches to fill the viewport."},
	}
)

func registerVersionPage(v *via.V) {
	v.Page("/version", func(c *via.Context) {
		var lastAction, lastError string
		settings, _ := getUISettingsFromNATS()

		// Helper to update settings
		update := func(field string, value string) {
			switch field {
			case "variant":
				settings.CSSVariant = value
			case "viewport":
				settings.ViewportMode = value
			}
			if err := setUISettingsInNATS(settings); err != nil {
				lastError = fmt.Sprintf("Failed to save: %v", err)
				lastAction = ""
			} else {
				lastAction = "Settings synced to all instances via NATS!"
				lastError = ""
			}
			c.Sync()
		}

		// Create variant actions dynamically
		variantActions := make(map[string]H)
		for _, opt := range cssVariants {
			val := opt.value
			variantActions[val] = c.Action(func() { update("variant", val) }).OnClick()
		}

		// Create viewport actions dynamically
		viewportActions := make(map[string]H)
		for _, opt := range viewportModes {
			val := opt.value
			viewportActions[val] = c.Action(func() { update("viewport", val) }).OnClick()
		}

		c.View(func() H {
			settingsMu.RLock()
			current := liveUISettings
			settingsMu.RUnlock()

			// Build variant buttons
			var variantBtns []H
			for _, opt := range cssVariants {
				class := "outline"
				if current.CSSVariant == opt.value {
					class = "contrast"
				}
				variantBtns = append(variantBtns, Button(Text(opt.label), Class(class), variantActions[opt.value]))
			}

			// Build viewport buttons
			var viewportBtns []H
			for _, opt := range viewportModes {
				class := "outline"
				if current.ViewportMode == opt.value {
					class = "contrast"
				}
				viewportBtns = append(viewportBtns, Button(Text(opt.label), Class(class), viewportActions[opt.value]))
			}

			// Build variant descriptions
			var variantDescs []H
			for _, opt := range cssVariants {
				variantDescs = append(variantDescs, Li(Strong(Text(opt.label+": ")), Text(opt.desc)))
			}

			// Build viewport descriptions
			var viewportDescs []H
			for _, opt := range viewportModes {
				viewportDescs = append(viewportDescs, Li(Strong(Text(opt.label+": ")), Text(opt.desc)))
			}

			var messageEl H
			if lastError != "" {
				messageEl = Article(Attr("data-theme", "light"), P(Class("pico-color-red"), Text(lastError)))
			} else if lastAction != "" {
				messageEl = Article(Attr("data-theme", "light"), P(Class("pico-color-green"), Text(lastAction)))
			}

			return Main(Class("container"),
				navBar("Version"),
				Section(
					H1(Text("Pico CSS Version Picker")),
					P(Text("Select the ideal Pico CSS variant to match your project's needs")),
					P(Text("NATS: "), natsStatusElement()),
				),
				messageEl,

				// CSS Variant
				Article(
					Header(H4(Text("CSS Variant"))),
					P(Text("Choose between class-based or semantic HTML styling:")),
					Div(append([]H{Role("group")}, variantBtns...)...),
					Details(Summary(Text("What's the difference?")), Ul(variantDescs...)),
				),

				// Viewport Mode
				Article(
					Header(H4(Text("Viewport Mode"))),
					P(Text("How should the layout behave?")),
					Div(append([]H{Role("group")}, viewportBtns...)...),
					Details(Summary(Text("Viewport modes explained")), Ul(viewportDescs...)),
				),

				// Current Config
				Article(
					Header(H5(Text("Current Configuration"))),
					Figure(Table(Role("grid"), TBody(
						Tr(Td(Strong(Text("CSS Variant"))), Td(Code(Text(current.CSSVariant)))),
						Tr(Td(Strong(Text("Viewport Mode"))), Td(Code(Text(current.ViewportMode)))),
						Tr(Td(Strong(Text("Theme"))), Td(func() H {
							if theme := getTheme(); theme != "" {
								return Code(Text(theme))
							}
							return Em(Text("(from env)"))
						}())),
					))),
				),

				// How It Works
				Section(Hr(), H5(Text("How It Works")),
					Ul(
						Li(Text("Settings stored in NATS KV bucket 'via_config' key 'ui_settings'")),
						Li(Text("All Via instances watch this key for changes")),
						Li(Text("Changes propagate instantly across all connected instances")),
						Li(Mark(Text("Note: ")), Text("Full CSS variant switching requires Via restart.")),
					),
				),

				// CDN Reference
				Section(
					H5(Text("Pico CSS CDN Reference")),
					P(Text("Based on your selection, use this CDN link:")),
					Pre(Code(Text(getCDNLink(current)))),
				),
			)
		})
	})
}

// getCDNLink returns the appropriate CDN link based on settings
func getCDNLink(settings UISettings) string {
	variant := "pico"
	if settings.CSSVariant == "classless" {
		variant = "pico.classless"
	}
	viewport := ""
	if settings.ViewportMode == "fluid" {
		viewport = ".fluid"
	}
	return fmt.Sprintf("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/%s%s.min.css", variant, viewport)
}
