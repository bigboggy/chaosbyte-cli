package memory

import (
	"context"
	"testing"

	"github.com/glycerine/hlc"

	"github.com/bchayka/gitstatus/internal/events"
	"github.com/bchayka/gitstatus/internal/ui"
)

func sampleActor() events.Actor {
	return events.Actor{ID: "pk:test", DisplayName: "@test", Kind: "human"}
}

func makeChat(t *testing.T, room, channel, body string, stamp int64) events.Event {
	t.Helper()
	e := events.NewChatPosted(room, sampleActor(), channel, body, ui.ChatNormal)
	e.SetStamp(events.Timestamp{HLC: hlc.HLC(stamp << 16)})
	return e
}

func TestMemoryAppendAndReplay(t *testing.T) {
	s := New()
	ctx := context.Background()

	for i, body := range []string{"first", "second", "third"} {
		if err := s.AppendEvent(ctx, makeChat(t, "vibespace", "#lobby", body, int64(i+1))); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
	}

	all, err := s.ReplayRoom(ctx, "vibespace", 0)
	if err != nil {
		t.Fatalf("ReplayRoom: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len = %d, want 3", len(all))
	}
	chat, ok := all[0].(*events.ChatPosted)
	if !ok || chat.Body != "first" {
		t.Errorf("first event = %+v", all[0])
	}
	chat, ok = all[2].(*events.ChatPosted)
	if !ok || chat.Body != "third" {
		t.Errorf("third event = %+v", all[2])
	}
}

func TestMemoryReplayLimit(t *testing.T) {
	s := New()
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_ = s.AppendEvent(ctx, makeChat(t, "vibespace", "#lobby", "msg", int64(i+1)))
	}
	got, _ := s.ReplayRoom(ctx, "vibespace", 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// Limited replay returns the latest events, ascending.
	first := got[0].(*events.ChatPosted)
	last := got[2].(*events.ChatPosted)
	if first.Timestamp().Before(last.Timestamp()) != true {
		t.Errorf("not ascending: %d vs %d", first.Timestamp().HLC, last.Timestamp().HLC)
	}
}

func TestMemoryLatestHLC(t *testing.T) {
	s := New()
	ctx := context.Background()
	ts, err := s.LatestHLC(ctx, "vibespace")
	if err != nil {
		t.Fatal(err)
	}
	if !ts.IsZero() {
		t.Errorf("empty room should return zero timestamp, got %d", ts.HLC)
	}

	_ = s.AppendEvent(ctx, makeChat(t, "vibespace", "#lobby", "hi", 5))
	_ = s.AppendEvent(ctx, makeChat(t, "vibespace", "#lobby", "hi", 2))
	_ = s.AppendEvent(ctx, makeChat(t, "vibespace", "#lobby", "hi", 7))

	ts, err = s.LatestHLC(ctx, "vibespace")
	if err != nil {
		t.Fatal(err)
	}
	if ts.HLC != 7<<16 {
		t.Errorf("LatestHLC = %d, want %d", ts.HLC, 7<<16)
	}
}

func TestMemoryRoomIsolation(t *testing.T) {
	s := New()
	ctx := context.Background()
	_ = s.AppendEvent(ctx, makeChat(t, "vibespace", "#lobby", "vibespace msg", 1))
	_ = s.AppendEvent(ctx, makeChat(t, "acme", "#lobby", "acme msg", 1))

	v, _ := s.ReplayRoom(ctx, "vibespace", 0)
	a, _ := s.ReplayRoom(ctx, "acme", 0)
	if len(v) != 1 || len(a) != 1 {
		t.Fatalf("room isolation broken: vibespace=%d acme=%d", len(v), len(a))
	}
}
