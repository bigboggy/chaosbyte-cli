// Package store is the persistence seam for the typed event bus. A
// Store records every event the broker publishes and serves replays
// to late joiners. Phase 1 ships two implementations: sqlite (the
// production backend, modernc.org/sqlite with WAL) and memory
// (in-process for tests and the local single-user binary).
//
// The interface is intentionally minimal in Phase 1: AppendEvent,
// ReplayRoom, LatestHLC. Layout snapshots and mod-state persistence
// land when the surfaces that need them go in (Phase 3 layout, Phase
// 1.1 mod cadence).
package store

import (
	"context"

	"github.com/bchayka/gitstatus/internal/events"
)

// Store is the persistence boundary for the event bus.
type Store interface {
	// AppendEvent records an event durably. The event must already
	// carry a non-zero HLC stamp; the broker is responsible for
	// stamping on Publish. AppendEvent is synchronous and returns
	// only when the event is fsync'd to disk (in WAL mode).
	AppendEvent(ctx context.Context, evt events.Event) error

	// ReplayRoom returns the most recent `limit` events for the room,
	// ascending by HLC. limit <= 0 returns every event.
	ReplayRoom(ctx context.Context, roomID string, limit int) ([]events.Event, error)

	// LatestHLC returns the highest HLC stamp persisted for the room,
	// or the zero timestamp if the room has no events yet. Used at
	// daemon boot to seed the in-memory clock past durable.
	LatestHLC(ctx context.Context, roomID string) (events.Timestamp, error)

	// Close releases any held resources (file handles, connections).
	Close() error
}
