package main

import (
	"fmt"
	"time"

	"github.com/go-via/via"
	. "github.com/go-via/via/h"
)

// registerChatPage registers the NATS chat page handler
func registerChatPage(v *via.V) {
	v.Page("/chat", func(c *via.Context) {
		userName := fmt.Sprintf("user-%d", time.Now().UnixNano()%1000)
		lastSent := ""

		// Create individual actions for each message
		sendHello := c.Action(func() {
			if err := sendChatMessage(userName, "Hello!"); err == nil {
				lastSent = "Hello!"
			}
			c.Sync()
		})
		sendHowAreYou := c.Action(func() {
			if err := sendChatMessage(userName, "How are you?"); err == nil {
				lastSent = "How are you?"
			}
			c.Sync()
		})
		sendNatsAwesome := c.Action(func() {
			if err := sendChatMessage(userName, "NATS is awesome!"); err == nil {
				lastSent = "NATS is awesome!"
			}
			c.Sync()
		})
		sendViaRocks := c.Action(func() {
			if err := sendChatMessage(userName, "Via rocks!"); err == nil {
				lastSent = "Via rocks!"
			}
			c.Sync()
		})

		// Subscribe to chat updates via NATS broadcast (no polling!)
		broadcast.Subscribe(TopicChat, func() {
			c.Sync()
		})

		c.View(func() H {
			chatMu.RLock()
			msgs := make([]ChatMessage, len(chatMessages))
			copy(msgs, chatMessages)
			chatMu.RUnlock()

			var msgList []H
			if len(msgs) == 0 {
				msgList = []H{Li(Em(Text("No messages yet. Click a button to say hi!")))}
			} else {
				for _, msg := range msgs {
					msgList = append(msgList, Li(
						Strong(Text(msg.From+": ")),
						Text(msg.Text),
						Small(Textf(" (%s)", msg.Time.Format("15:04:05"))),
					))
				}
			}

			var sentMsg H
			if lastSent != "" {
				sentMsg = P(Small(Text("Sent: " + lastSent)))
			}

			return Main(Class("container"),
				navBar("Chat"),

				Section(
					H1(Text("NATS Chat")),
					P(Text("Cross-instance messaging via NATS pub/sub")),
					P(Text("Status: "), natsStatusElement(), Text(" | You are: "), Strong(Text(userName))),
				),

				Article(
					Header(H4(Text("Messages"))),
					Ul(msgList...),
				),

				Section(
					H5(Text("Send a message:")),
					Div(Role("group"),
						Button(Text("Hello!"), sendHello.OnClick()),
						Button(Text("How are you?"), sendHowAreYou.OnClick()),
						Button(Text("NATS is awesome!"), sendNatsAwesome.OnClick()),
						Button(Text("Via rocks!"), sendViaRocks.OnClick()),
					),
					sentMsg,
				),

				Section(
					Hr(),
					H5(Text("How It Works")),
					Ul(
						Li(Text("Messages are published to NATS subject 'via.chat'")),
						Li(Text("All Via instances subscribe to this subject")),
						Li(Text("Open multiple browser tabs - they all see the same messages!")),
					),
				),
			)
		})
	})
}
