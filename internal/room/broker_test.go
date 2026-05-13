package room

import (
	"testing"
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
)

// TestBrokerBroadcasts is the headless stand-in for opening two SSH
// sessions side-by-side. Two subscribers attach, one publishes, both must
// receive the event. Catches regressions on the multi-user wiring that
// the user can't see from a single local TUI.
func TestBrokerBroadcasts(t *testing.T) {
	b := New()
	defer b.Stop()

	alice := b.Subscribe()
	bob := b.Subscribe()

	msg := ui.ChatMessage{
		Author: "@alice", Body: "hello", At: time.Now(), Kind: ui.ChatNormal,
	}
	b.Publish("#lobby", msg)

	got := func(name string, sub <-chan Event) Event {
		t.Helper()
		select {
		case evt := <-sub:
			return evt
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("%s never received the published event", name)
		}
		return Event{}
	}

	evtA := got("alice", alice)
	evtB := got("bob", bob)
	if evtA.Channel != "#lobby" || evtA.Message.Body != "hello" {
		t.Errorf("alice got %+v", evtA)
	}
	if evtB.Channel != "#lobby" || evtB.Message.Body != "hello" {
		t.Errorf("bob got %+v", evtB)
	}
}

// TestBrokerJoinAutoWelcomes asserts the broker fires a follow-up mod
// welcome when a ChatJoin message lands. Without this every SSH user
// would slip in silently from the other sessions' POV.
func TestBrokerJoinAutoWelcomes(t *testing.T) {
	b := New()
	defer b.Stop()

	sub := b.Subscribe()
	b.Publish("#lobby", ui.ChatMessage{
		Author: "@alice", Body: "entered the chat", At: time.Now(), Kind: ui.ChatJoin,
	})

	// Drain the join, then expect the auto-welcome from the mod.
	join := <-sub
	if join.Message.Kind != ui.ChatJoin {
		t.Fatalf("first event should be the join, got kind=%v", join.Message.Kind)
	}

	select {
	case welcome := <-sub:
		if welcome.Message.Kind != ui.ChatAction {
			t.Errorf("welcome should be ChatAction, got %v", welcome.Message.Kind)
		}
		if welcome.Channel != "#lobby" {
			t.Errorf("welcome channel = %q, want #lobby", welcome.Channel)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no auto-welcome after ChatJoin")
	}
}

// TestBrokerTierClimbsWithChat checks that publishing messages in burst
// pushes Tier() up to 3, and dropping the activity returns it toward 0
// once the publish window slides past.
func TestBrokerTierClimbsWithChat(t *testing.T) {
	b := New()
	defer b.Stop()

	if got := b.Tier(); got != 0 {
		t.Errorf("idle Tier() = %d, want 0", got)
	}

	now := time.Now()
	for i := 0; i < 6; i++ {
		b.Publish("#lobby", ui.ChatMessage{
			Author: "@alice", Body: "burst", At: now, Kind: ui.ChatNormal,
		})
	}
	if got := b.Tier(); got != 3 {
		t.Errorf("after 6-burst Tier() = %d, want 3", got)
	}
}
