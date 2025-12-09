package main

import (
	"time"

	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// Note: page_monitor needs time for stats calculation, not for polling

// registerMonitorPage registers the NATS message monitor page handler
func registerMonitorPage(v *via.V) {
	v.Page("/monitor", func(c *via.Context) {
		currentPattern := ">"
		isSubscribed := false
		var lastError string
		var lastAction string

		// Subscribe action
		subscribeAll := c.Action(func() {
			currentPattern = ">"
			if err := startMonitorSubscription(">"); err != nil {
				lastError = err.Error()
				lastAction = ""
				isSubscribed = false
			} else {
				lastError = ""
				lastAction = "Subscribed to all subjects (>)"
				isSubscribed = true
			}
			c.Sync()
		})

		subscribeVia := c.Action(func() {
			currentPattern = "via.>"
			if err := startMonitorSubscription("via.>"); err != nil {
				lastError = err.Error()
				lastAction = ""
				isSubscribed = false
			} else {
				lastError = ""
				lastAction = "Subscribed to via.> subjects"
				isSubscribed = true
			}
			c.Sync()
		})

		subscribeKV := c.Action(func() {
			currentPattern = "$KV.>"
			if err := startMonitorSubscription("$KV.>"); err != nil {
				lastError = err.Error()
				lastAction = ""
				isSubscribed = false
			} else {
				lastError = ""
				lastAction = "Subscribed to $KV.> (KV changes)"
				isSubscribed = true
			}
			c.Sync()
		})

		unsubscribe := c.Action(func() {
			stopMonitorSubscription()
			lastError = ""
			lastAction = "Unsubscribed"
			isSubscribed = false
			c.Sync()
		})

		clearMessages := c.Action(func() {
			clearMonitorMessages()
			lastAction = "Messages cleared"
			c.Sync()
		})

		// Subscribe to monitor updates via NATS broadcast (no polling!)
		broadcast.Subscribe(TopicMonitor, func() {
			c.Sync()
		})

		c.View(func() H {
			monitorMu.RLock()
			subActive := monitorSub != nil
			monitorMu.RUnlock()

			// Sync state
			isSubscribed = subActive

			// Get messages and stats
			msgs := getMonitorMessages()
			stats := getMonitorStats()

			var messageEl H
			if lastError != "" {
				messageEl = Article(Attr("data-theme", "light"),
					P(Class("pico-color-red"), Text(lastError)),
				)
			} else if lastAction != "" {
				messageEl = Article(Attr("data-theme", "light"),
					P(Class("pico-color-green"), Text(lastAction)),
				)
			}

			// Stats section
			var rate float64
			if stats.TotalMessages > 0 && !stats.StartTime.IsZero() {
				elapsed := time.Since(stats.StartTime).Seconds()
				if elapsed > 0 {
					rate = float64(stats.TotalMessages) / elapsed
				}
			}

			statsEl := Article(
				Header(H4(Text("Statistics"))),
				Div(Role("group"),
					Div(
						Strong(Text("Total Messages: ")),
						Code(Textf("%d", stats.TotalMessages)),
					),
					Div(
						Strong(Text("Rate: ")),
						Code(Textf("%.1f/sec", rate)),
					),
					Div(
						Strong(Text("Subjects Seen: ")),
						Code(Textf("%d", len(stats.SubjectsSeen))),
					),
				),
			)

			// Message log (most recent first)
			var msgRows []H
			if len(msgs) == 0 {
				msgRows = []H{
					Tr(Td(Attr("colspan", "4"), Em(Text("No messages captured yet. Click a subscribe button to start monitoring.")))),
				}
			} else {
				// Show messages in reverse order (newest first)
				for i := len(msgs) - 1; i >= 0; i-- {
					msg := msgs[i]
					// Truncate data if very long for display
					displayData := msg.Data
					if len(displayData) > 80 {
						displayData = displayData[:80] + "..."
					}
					msgRows = append(msgRows, Tr(
						Td(Small(Text(msg.Time.Format("15:04:05.000")))),
						Td(Strong(Code(Text(msg.Subject)))),
						Td(Textf("%d", msg.Size)),
						Td(Small(Text(displayData))),
					))
				}
			}

			// Subjects breakdown
			var subjectList []H
			for subject, count := range stats.SubjectsSeen {
				subjectList = append(subjectList, Li(
					Code(Text(subject)),
					Text(": "),
					Strong(Textf("%d", count)),
				))
			}
			if len(subjectList) == 0 {
				subjectList = []H{Li(Em(Text("No subjects seen yet")))}
			}

			return Main(Class("container"),
				navBar("Monitor"),

				Section(
					H1(Text("NATS Message Monitor")),
					P(Text("Real-time view of NATS messages flowing through the system")),
					P(Text("NATS: "), natsStatusElement()),
				),

				messageEl,

				Section(
					H4(Text("Subscribe to Subjects")),
					P(Small(Text("Pattern: "), Code(Text(currentPattern)), Text(" | Status: "),
						func() H {
							if isSubscribed {
								return Ins(Text("Subscribed"))
							}
							return Del(Text("Not subscribed"))
						}(),
					)),
					Div(Role("group"),
						Button(Text("All (>)"), func() H {
							if isSubscribed && currentPattern == ">" {
								return Class("secondary")
							}
							return nil
						}(), subscribeAll.OnClick()),
						Button(Text("via.>"), func() H {
							if isSubscribed && currentPattern == "via.>" {
								return Class("secondary")
							}
							return nil
						}(), subscribeVia.OnClick()),
						Button(Text("$KV.>"), func() H {
							if isSubscribed && currentPattern == "$KV.>" {
								return Class("secondary")
							}
							return nil
						}(), subscribeKV.OnClick()),
						Button(Text("Stop"), Class("outline secondary"), unsubscribe.OnClick()),
						Button(Text("Clear"), Class("outline contrast"), clearMessages.OnClick()),
					),
				),

				statsEl,

				Section(
					H4(Text("Message Log")),
					P(Small(Textf("Showing last %d messages (newest first)", len(msgs)))),
					Figure(
						Table(Role("grid"),
							THead(Tr(
								Th(Text("Time")),
								Th(Text("Subject")),
								Th(Text("Size")),
								Th(Text("Data")),
							)),
							TBody(msgRows...),
						),
					),
				),

				Section(
					H5(Text("Subjects Seen")),
					Ul(subjectList...),
				),
			)
		})
	})
}
