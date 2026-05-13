package events

import (
	"encoding/json"
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
)

const kindChatPosted = "chat.posted"

// ChatPosted is fired when any participant posts a message in a channel.
// Body, MessageKind, and At are carried verbatim from ui.ChatMessage so
// existing rendering paths continue to work.
type ChatPosted struct {
	*Header

	Channel     string         `json:"channel"`
	Body        string         `json:"body"`
	MessageKind ui.ChatKind    `json:"message_kind"`
	At          time.Time      `json:"at"`
}

// chatPostedPayload is the JSON wire shape that lives inside the
// envelope's "payload" field.
type chatPostedPayload struct {
	Channel     string      `json:"channel"`
	Body        string      `json:"body"`
	MessageKind ui.ChatKind `json:"message_kind"`
	At          time.Time   `json:"at"`
}

func (c *ChatPosted) EventKind() string { return kindChatPosted }

func (c *ChatPosted) MarshalPayload() (json.RawMessage, error) {
	return json.Marshal(chatPostedPayload{
		Channel:     c.Channel,
		Body:        c.Body,
		MessageKind: c.MessageKind,
		At:          c.At,
	})
}

// NewChatPosted constructs a ChatPosted event with the Header filled.
// The broker assigns ID and Stamp on Publish.
func NewChatPosted(room string, actor Actor, channel, body string, kind ui.ChatKind) *ChatPosted {
	return &ChatPosted{
		Header:      NewHeader(room, actor),
		Channel:     channel,
		Body:        body,
		MessageKind: kind,
		At:          time.Now(),
	}
}

// AsChatMessage converts the event back into the existing ui.ChatMessage
// shape used by the lobby renderer. Author is taken from the Actor's
// display name.
func (c *ChatPosted) AsChatMessage() ui.ChatMessage {
	return ui.ChatMessage{
		Author: c.Actor.DisplayName,
		Body:   c.Body,
		At:     c.At,
		Kind:   c.MessageKind,
	}
}
