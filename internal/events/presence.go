package events

import "encoding/json"

const (
	kindPresenceJoined = "presence.joined"
	kindPresenceLeft   = "presence.left"
)

// PresenceJoined is fired when a session subscribes to a room. The
// DisplayName is copied from the Actor; consumers can use it without
// resolving the principal.
type PresenceJoined struct {
	*Header
	DisplayName string `json:"display_name"`
}

type presenceJoinedPayload struct {
	DisplayName string `json:"display_name"`
}

func (p *PresenceJoined) EventKind() string { return kindPresenceJoined }

func (p *PresenceJoined) MarshalPayload() (json.RawMessage, error) {
	return json.Marshal(presenceJoinedPayload{DisplayName: p.DisplayName})
}

// NewPresenceJoined builds a PresenceJoined event with the Header
// filled. The broker assigns ID and Stamp on Publish.
func NewPresenceJoined(room string, actor Actor) *PresenceJoined {
	return &PresenceJoined{
		Header:      NewHeader(room, actor),
		DisplayName: actor.DisplayName,
	}
}

// PresenceLeft is fired when a session unsubscribes from a room.
// Reason values: "quit", "disconnect", "kicked", "stalled".
type PresenceLeft struct {
	*Header
	Reason string `json:"reason"`
}

type presenceLeftPayload struct {
	Reason string `json:"reason"`
}

func (p *PresenceLeft) EventKind() string { return kindPresenceLeft }

func (p *PresenceLeft) MarshalPayload() (json.RawMessage, error) {
	return json.Marshal(presenceLeftPayload{Reason: p.Reason})
}

// NewPresenceLeft builds a PresenceLeft event.
func NewPresenceLeft(room string, actor Actor, reason string) *PresenceLeft {
	return &PresenceLeft{
		Header: NewHeader(room, actor),
		Reason: reason,
	}
}
