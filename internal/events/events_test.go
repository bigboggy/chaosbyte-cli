package events

import (
	"testing"
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/google/uuid"
)

func sampleActor() Actor {
	return Actor{
		ID:          "pk:abc123",
		DisplayName: "@daniel",
		Kind:        "human",
		SessionID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	}
}

// TestChatPostedRoundtrip confirms a ChatPosted event survives Marshal +
// Unmarshal with all fields preserved.
func TestChatPostedRoundtrip(t *testing.T) {
	at := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	e := NewChatPosted("vibespace", sampleActor(), "lobby", "hello world", ui.ChatNormal)
	e.SetID(uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	e.SetStamp(Timestamp{HLC: 1234567890})
	e.At = at

	data, err := Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	chat, ok := got.(*ChatPosted)
	if !ok {
		t.Fatalf("Unmarshal returned %T, want *ChatPosted", got)
	}
	if chat.EventKind() != kindChatPosted {
		t.Errorf("EventKind = %q, want %q", chat.EventKind(), kindChatPosted)
	}
	if chat.EventID() != e.EventID() {
		t.Errorf("EventID mismatch")
	}
	if chat.Room() != "vibespace" {
		t.Errorf("Room = %q", chat.Room())
	}
	if chat.ActorRef().DisplayName != "@daniel" {
		t.Errorf("Actor display = %q", chat.ActorRef().DisplayName)
	}
	if chat.Channel != "lobby" {
		t.Errorf("Channel = %q", chat.Channel)
	}
	if chat.Body != "hello world" {
		t.Errorf("Body = %q", chat.Body)
	}
	if !chat.At.Equal(at) {
		t.Errorf("At mismatch: %v vs %v", chat.At, at)
	}
	if int64(chat.Timestamp().HLC) != 1234567890 {
		t.Errorf("Stamp mismatch: %d", int64(chat.Timestamp().HLC))
	}
}

func TestPresenceJoinedRoundtrip(t *testing.T) {
	e := NewPresenceJoined("vibespace", sampleActor())
	e.SetID(uuid.New())
	e.SetStamp(Timestamp{HLC: 100})
	data, err := Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	pj, ok := got.(*PresenceJoined)
	if !ok {
		t.Fatalf("Unmarshal returned %T, want *PresenceJoined", got)
	}
	if pj.DisplayName != "@daniel" {
		t.Errorf("DisplayName = %q", pj.DisplayName)
	}
}

func TestPresenceLeftRoundtrip(t *testing.T) {
	e := NewPresenceLeft("vibespace", sampleActor(), "quit")
	e.SetID(uuid.New())
	e.SetStamp(Timestamp{HLC: 101})
	data, err := Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	pl, ok := got.(*PresenceLeft)
	if !ok {
		t.Fatalf("Unmarshal returned %T, want *PresenceLeft", got)
	}
	if pl.Reason != "quit" {
		t.Errorf("Reason = %q", pl.Reason)
	}
}

func TestModTaggedRoundtrip(t *testing.T) {
	target := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	e := NewModTagged("vibespace", sampleActor(), target, "?", "question")
	e.SetID(uuid.New())
	e.SetStamp(Timestamp{HLC: 200})
	data, err := Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	mt, ok := got.(*ModTagged)
	if !ok {
		t.Fatalf("Unmarshal returned %T, want *ModTagged", got)
	}
	if mt.TargetEventID != target {
		t.Errorf("TargetEventID mismatch")
	}
	if mt.Marker != "?" || mt.Reason != "question" {
		t.Errorf("Marker / Reason mismatch")
	}
}

// TestUnknownKindPreservesPayload confirms a future topic that this
// build does not recognize is returned as Unknown with the raw payload
// preserved, so an older client can still process the event.
func TestUnknownKindPreservesPayload(t *testing.T) {
	raw := []byte(`{
		"kind": "future.topic",
		"id": "44444444-4444-4444-4444-444444444444",
		"room": "vibespace",
		"actor": {"id":"pk:x","display":"@x","kind":"human","session":"11111111-1111-1111-1111-111111111111"},
		"stamp": 500,
		"payload": {"foo": "bar"}
	}`)
	got, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	u, ok := got.(*Unknown)
	if !ok {
		t.Fatalf("Unmarshal returned %T, want *Unknown", got)
	}
	if u.EventKind() != "future.topic" {
		t.Errorf("Kind = %q", u.EventKind())
	}
	if u.Room() != "vibespace" {
		t.Errorf("Room = %q", u.Room())
	}
	if string(u.RawPayload) != `{"foo": "bar"}` {
		t.Errorf("RawPayload = %q", string(u.RawPayload))
	}
}

// TestClockAdvances confirms Now produces strictly increasing stamps.
func TestClockAdvances(t *testing.T) {
	c := NewClock()
	t1 := c.Now()
	t2 := c.Now()
	if !t2.After(t1) {
		t.Errorf("t2 should be after t1: %d <= %d", int64(t2.HLC), int64(t1.HLC))
	}
}

// TestClockUpdate confirms Update merges a remote stamp such that the
// next local Now produces a stamp later than both.
func TestClockUpdate(t *testing.T) {
	c := NewClock()
	local := c.Now()

	// Remote is artificially in the future by bumping the wall portion.
	remote := Timestamp{HLC: local.HLC + (60_000_000_000 << 16)}
	_ = c.Update(remote)
	next := c.Now()
	if !next.After(remote) {
		t.Errorf("next should be after remote: %d <= %d", int64(next.HLC), int64(remote.HLC))
	}
}

// TestTimestampSerialization confirms Timestamp JSON marshaling
// round-trips through json.Marshal/json.Unmarshal and the wire form is
// stable (number, not object).
func TestTimestampSerialization(t *testing.T) {
	ts := Timestamp{HLC: 1234567890}
	data, err := ts.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != "1234567890" {
		t.Errorf("wire form = %s, want bare number", string(data))
	}
	var got Timestamp
	if err := got.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if got.HLC != ts.HLC {
		t.Errorf("roundtrip mismatch: %d != %d", int64(got.HLC), int64(ts.HLC))
	}
}

// TestTimestampBytes confirms 8-byte BigEndian wire form for the store.
func TestTimestampBytes(t *testing.T) {
	ts := Timestamp{HLC: 0x0123456789ABCDEF}
	b := ts.Bytes()
	if len(b) != 8 {
		t.Fatalf("Bytes length = %d, want 8", len(b))
	}
	got, err := FromBytes(b)
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	if got.HLC != ts.HLC {
		t.Errorf("roundtrip mismatch")
	}
}
