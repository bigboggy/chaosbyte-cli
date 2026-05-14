package sqlite

import (
	"context"
	"path/filepath"
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

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSQLiteOpenAppliesMigrations(t *testing.T) {
	s := newTestStore(t)
	var version int
	err := s.db.QueryRow(`SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err != nil {
		t.Fatalf("migrations not applied: %v", err)
	}
	if version < 1 {
		t.Errorf("schema version = %d, want >= 1", version)
	}
}

func TestSQLiteWALMode(t *testing.T) {
	s := newTestStore(t)
	var mode string
	if err := s.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func TestSQLiteAppendAndReplay(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for i, body := range []string{"first", "second", "third"} {
		if err := s.AppendEvent(ctx, makeChat(t, "vibespace", "#lobby", body, int64(i+1))); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
	}

	all, err := s.ReplayRoom(ctx, "vibespace", 0)
	if err != nil {
		t.Fatal(err)
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

func TestSQLiteReplayLimit(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_ = s.AppendEvent(ctx, makeChat(t, "vibespace", "#lobby", "msg", int64(i+1)))
	}
	got, _ := s.ReplayRoom(ctx, "vibespace", 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	first := got[0].(*events.ChatPosted)
	last := got[2].(*events.ChatPosted)
	if !first.Timestamp().Before(last.Timestamp()) {
		t.Errorf("ascending order broken")
	}
	// The most recent 3 events have stamps 8, 9, 10.
	if last.Timestamp().HLC != 10<<16 {
		t.Errorf("last stamp = %d, want %d", last.Timestamp().HLC, 10<<16)
	}
}

func TestSQLiteLatestHLC(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	ts, err := s.LatestHLC(ctx, "vibespace")
	if err != nil {
		t.Fatal(err)
	}
	if !ts.IsZero() {
		t.Errorf("empty room should return zero, got %d", ts.HLC)
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

func TestSQLitePersistsAcrossOpens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist.db")

	{
		s, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		_ = s.AppendEvent(context.Background(), makeChat(t, "vibespace", "#lobby", "persisted", 42))
		s.Close()
	}

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.ReplayRoom(context.Background(), "vibespace", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("after reopen len = %d, want 1", len(got))
	}
	chat, ok := got[0].(*events.ChatPosted)
	if !ok || chat.Body != "persisted" {
		t.Errorf("recovered event = %+v", got[0])
	}
}

func TestSQLiteMigrationsAreIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "idempotent.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.Close()

	// Re-open: should not re-apply.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer s2.Close()
	var count int
	if err := s2.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("schema_migrations rows = %d, want 1", count)
	}
}
