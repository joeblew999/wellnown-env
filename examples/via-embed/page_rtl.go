package main

import (
	"fmt"

	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// RTL language options with sample text
var rtlLanguages = []struct {
	Code, Name, Sample string
}{
	{"ar", "Arabic", "مرحبا بكم في التطبيق"},
	{"he", "Hebrew", "ברוכים הבאים לאפליקציה"},
	{"fa", "Persian (Farsi)", "به برنامه خوش آمدید"},
	{"ur", "Urdu", "ایپلیکیشن میں خوش آمدید"},
}

func registerRTLPage(v *via.V) {
	v.Page("/rtl", func(c *via.Context) {
		var lastAction, lastError string
		settings, _ := getUISettingsFromNATS()

		update := func() {
			if err := setUISettingsInNATS(settings); err != nil {
				lastError = fmt.Sprintf("Failed to save: %v", err)
				lastAction = ""
			} else {
				lastAction = "RTL settings synced to all instances via NATS!"
				lastError = ""
			}
			c.Sync()
		}

		toggleRTL := c.Action(func() {
			settings.RTLEnabled = !settings.RTLEnabled
			update()
		})

		// Dynamic language actions
		langActions := make(map[string]H)
		for _, lang := range rtlLanguages {
			code := lang.Code
			langActions[code] = c.Action(func() {
				settings.RTLLang = code
				settings.RTLEnabled = true
				update()
			}).OnClick()
		}

		c.View(func() H {
			settingsMu.RLock()
			cur := liveUISettings
			settingsMu.RUnlock()

			// Find current language info
			curSample, curLangName := rtlLanguages[0].Sample, rtlLanguages[0].Name
			for _, lang := range rtlLanguages {
				if lang.Code == cur.RTLLang {
					curSample, curLangName = lang.Sample, lang.Name
					break
				}
			}

			dir := "ltr"
			if cur.RTLEnabled {
				dir = "rtl"
			}

			// Message element
			var messageEl H
			if lastError != "" {
				messageEl = Article(Attr("data-theme", "light"), P(Class("pico-color-red"), Text(lastError)))
			} else if lastAction != "" {
				messageEl = Article(Attr("data-theme", "light"), P(Class("pico-color-green"), Text(lastAction)))
			}

			// Language buttons
			var langBtns []H
			for _, lang := range rtlLanguages {
				class := "outline"
				if cur.RTLLang == lang.Code {
					class = "contrast"
				}
				langBtns = append(langBtns, Button(Text(lang.Name), Class(class), langActions[lang.Code]))
			}

			// Language table rows
			var langRows []H
			for _, lang := range rtlLanguages {
				langRows = append(langRows, Tr(
					Td(Code(Text(lang.Code))),
					Td(Text(lang.Name)),
					Td(Span(Attr("dir", "rtl"), Attr("lang", lang.Code), Text(lang.Sample))),
				))
			}

			return Main(Class("container"),
				navBar("RTL"),
				Section(
					H1(Text("RTL (Right-to-Left) Support")),
					P(Text("Enable right-to-left text rendering for Arabic, Hebrew, Persian, and more")),
					P(Text("NATS: "), natsStatusElement()),
				),
				messageEl,

				// RTL Toggle
				Article(
					Header(H4(Text("RTL Mode"))),
					P(Text("Toggle right-to-left text direction:")),
					Div(Role("group"), Button(
						Text(func() string {
							if cur.RTLEnabled {
								return "Enabled (RTL)"
							}
							return "Disabled (LTR)"
						}()),
						Class(func() string {
							if cur.RTLEnabled {
								return "contrast"
							}
							return "outline secondary"
						}()),
						toggleRTL.OnClick(),
					)),
					P(Small(Text("Current: "), Code(Textf("dir=\"%s\"", dir)))),
				),

				// Language Selection
				Article(
					Header(H4(Text("RTL Language"))),
					P(Text("Select a language to see RTL in action:")),
					Div(append([]H{Role("group")}, langBtns...)...),
				),

				// Live Demo
				Article(
					Header(H4(Text("Live RTL Demo"))),
					func() H {
						if cur.RTLEnabled {
							return Div(
								BlockQuote(Attr("dir", "rtl"), Attr("lang", cur.RTLLang),
									P(Text(curSample)),
									Footer(Cite(Text("- "+curLangName+" Sample Text"))),
								),
								P(Text("Mixed content example:")),
								P(Attr("dir", "rtl"), Attr("lang", cur.RTLLang),
									Text(curSample+" "), Strong(Text("("+curLangName+")")),
									Text(" - This demonstrates RTL text flow."),
								),
							)
						}
						return BlockQuote(
							P(Text("Enable RTL mode above to see right-to-left text rendering")),
							Footer(Cite(Text("- Demo placeholder"))),
						)
					}(),
				),

				// Current Config
				Article(
					Header(H5(Text("Current Configuration"))),
					Figure(Table(Role("grid"), TBody(
						Tr(Td(Strong(Text("RTL Enabled"))), Td(Code(Textf("%v", cur.RTLEnabled)))),
						Tr(Td(Strong(Text("Language"))), Td(Code(Text(cur.RTLLang+" ("+curLangName+")")))),
						Tr(Td(Strong(Text("HTML Attribute"))), Td(Code(Textf("dir=\"%s\" lang=\"%s\"", dir, cur.RTLLang)))),
					))),
				),

				// How It Works
				Section(Hr(), H5(Text("How RTL Works")),
					Ul(
						Li(Text("Add "), Code(Text("dir=\"rtl\"")), Text(" to the HTML element for global RTL")),
						Li(Text("Or apply to individual elements for selective RTL")),
						Li(Text("Use "), Code(Text("lang")), Text(" attribute for proper font selection")),
						Li(Text("Pico CSS handles RTL styling automatically")),
					),
				),

				// Implementation Examples
				Section(H5(Text("Implementation Examples")),
					P(Strong(Text("Global RTL:"))),
					Pre(Code(Text(`<html dir="rtl" lang="ar">...</html>`))),
					P(Strong(Text("Per-element RTL:"))),
					Pre(Code(Text(`<blockquote dir="rtl" lang="ar">مرحبا</blockquote>`))),
				),

				// Language Table
				Section(H5(Text("Common RTL Languages")),
					Figure(Table(Role("grid"),
						THead(Tr(Th(Text("Code")), Th(Text("Language")), Th(Text("Sample")))),
						TBody(langRows...),
					)),
				),
			)
		})
	})
}
