package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// subscribeToChatMessages subscribes to the chat subject
func subscribeToChatMessages() {
	nc, err := getNatsConn()
	if err != nil {
		return
	}

	sub, err := nc.Subscribe("via.chat", func(msg *nats.Msg) {
		var chatMsg ChatMessage
		if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
			return
		}
		chatMu.Lock()
		chatMessages = append(chatMessages, chatMsg)
		// Keep last 50 messages
		if len(chatMessages) > 50 {
			chatMessages = chatMessages[len(chatMessages)-50:]
		}
		chatMu.Unlock()
		fmt.Printf("[CHAT] %s: %s\n", chatMsg.From, chatMsg.Text)
		// Notify all subscribed clients
		broadcast.Notify(TopicChat)
	})
	if err != nil {
		fmt.Printf("Error subscribing to chat: %v\n", err)
		return
	}
	chatSub = sub
}

// sendChatMessage publishes a chat message via NATS
func sendChatMessage(from, text string) error {
	nc, err := getNatsConn()
	if err != nil {
		return err
	}

	msg := ChatMessage{
		From: from,
		Text: text,
		Time: time.Now(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return nc.Publish("via.chat", data)
}
