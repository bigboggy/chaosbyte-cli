// Package memory provides an in-process Store implementation. Used by
// tests and the local single-user binary (cmd/vibespace). Not durable
// across restarts.
package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/bchayka/gitstatus/internal/events"
	"github.com/bchayka/gitstatus/internal/store"
)

// Store is the in-memory implementation of store.Store. Safe for
// concurrent use.
type Store struct {
	mu     sync.RWMutex
	byRoom map[string][]events.Event
}

// New returns a fresh in-memory store.
func New() *Store {
	return &Store{byRoom: map[string][]events.Event{}}
}

// AppendEvent records an event. Events are kept in publish order; the
// caller is responsible for HLC stamping before AppendEvent runs.
func (s *Store) AppendEvent(_ context.Context, evt events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	room := evt.Room()
	s.byRoom[room] = append(s.byRoom[room], evt)
	return nil
}

// ReplayRoom returns the most recent `limit` events for the room,
// ascending by HLC. limit <= 0 returns every event.
func (s *Store) ReplayRoom(_ context.Context, roomID string, limit int) ([]events.Event, error) {
	s.mu.RLock()
	src := s.byRoom[roomID]
	out := make([]events.Event, len(src))
	copy(out, src)
	s.mu.RUnlock()

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Timestamp().Before(out[j].Timestamp())
	})

	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

// LatestHLC returns the highest HLC stamp persisted for the room.
func (s *Store) LatestHLC(_ context.Context, roomID string) (events.Timestamp, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var max events.Timestamp
	for _, evt := range s.byRoom[roomID] {
		if evt.Timestamp().After(max) {
			max = evt.Timestamp()
		}
	}
	return max, nil
}

// Close is a no-op for the in-memory store.
func (s *Store) Close() error { return nil }

// Assert *Store satisfies store.Store at compile time.
var _ store.Store = (*Store)(nil)
