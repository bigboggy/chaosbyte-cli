package events

import (
	"encoding/json"

	"github.com/google/uuid"
)

const kindModTagged = "mod.tagged"

// ModTagged is fired when the moderator attaches a tag (e.g., a "?"
// marker for a question) to an existing event. The TargetEventID
// points at the originating event; the marker is the glyph rendered
// in the chat margin.
type ModTagged struct {
	*Header
	TargetEventID uuid.UUID `json:"target_event"`
	Marker        string    `json:"marker"`
	Reason        string    `json:"reason"`
}

type modTaggedPayload struct {
	TargetEventID uuid.UUID `json:"target_event"`
	Marker        string    `json:"marker"`
	Reason        string    `json:"reason"`
}

func (m *ModTagged) EventKind() string { return kindModTagged }

func (m *ModTagged) MarshalPayload() (json.RawMessage, error) {
	return json.Marshal(modTaggedPayload{
		TargetEventID: m.TargetEventID,
		Marker:        m.Marker,
		Reason:        m.Reason,
	})
}

// NewModTagged builds a ModTagged event.
func NewModTagged(room string, actor Actor, target uuid.UUID, marker, reason string) *ModTagged {
	return &ModTagged{
		Header:        NewHeader(room, actor),
		TargetEventID: target,
		Marker:        marker,
		Reason:        reason,
	}
}
