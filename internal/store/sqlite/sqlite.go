// Package sqlite is the SQLite-backed Store implementation. WAL mode
// + synchronous=NORMAL is the standard pair for a single-writer
// workload: durable across daemon crashes without the 5x cost of
// synchronous=FULL.
//
// The driver is modernc.org/sqlite, registered as "sqlite". Pure Go,
// no cgo, cross-compiles statically.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/glycerine/hlc"

	"github.com/bchayka/gitstatus/internal/events"
	"github.com/bchayka/gitstatus/internal/store"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Store is the SQLite-backed event log.
type Store struct {
	db   *sql.DB
	path string
}

// Open returns a Store backed by SQLite at the given path. Creates the
// file and applies migrations if it does not exist. WAL mode is
// enabled via URL parameters.
func Open(path string) (*Store, error) {
	// URL params:
	//   _pragma=journal_mode(WAL)       durable, single-writer
	//   _pragma=synchronous(NORMAL)     skip the FULL fsync per commit
	//   _pragma=foreign_keys(ON)        defensive
	//   _pragma=busy_timeout(5000)      5s on lock contention
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: ping %s: %w", path, err)
	}
	s := &Store{db: db, path: path}
	if err := s.applyMigrations(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// applyMigrations runs every embedded migration whose version is newer
// than what schema_migrations records. Each migration runs in its own
// transaction.
func (s *Store) applyMigrations() error {
	// Discover available migrations in numeric order.
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("sqlite: read migrations: %w", err)
	}
	type mig struct {
		version int
		name    string
	}
	var migs []mig
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		// File name: NNNN_description.sql
		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		var v int
		if _, err := fmt.Sscanf(parts[0], "%d", &v); err != nil {
			continue
		}
		migs = append(migs, mig{version: v, name: e.Name()})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })

	// Bootstrap: the migration table itself lives in the first
	// migration, so the first run is special-cased.
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("sqlite: bootstrap migrations table: %w", err)
	}

	applied := map[int]bool{}
	rows, err := s.db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("sqlite: read applied migrations: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()

	for _, m := range migs {
		if applied[m.version] {
			continue
		}
		body, err := migrationFS.ReadFile(filepath.Join("migrations", m.name))
		if err != nil {
			return fmt.Errorf("sqlite: read %s: %w", m.name, err)
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("sqlite: begin migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("sqlite: apply %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("sqlite: commit %s: %w", m.name, err)
		}
	}
	return nil
}

// AppendEvent records the event in the log. The event must already
// carry a non-zero HLC stamp; the broker stamps before calling this.
func (s *Store) AppendEvent(ctx context.Context, evt events.Event) error {
	if evt.Timestamp().IsZero() {
		return errors.New("sqlite: event timestamp is zero; broker should stamp before AppendEvent")
	}

	body, err := events.Marshal(evt)
	if err != nil {
		return fmt.Errorf("sqlite: marshal event: %w", err)
	}

	wall, logical := splitHLC(evt.Timestamp().HLC)
	actor := evt.ActorRef()
	idBytes, _ := evt.EventID().MarshalBinary()
	sessionBytes, _ := actor.SessionID.MarshalBinary()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO events
			(room_id, hlc_wall, hlc_logical, event_id, kind,
			 actor_id, actor_kind, session_id, capability_raw, envelope_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		evt.Room(), wall, logical, idBytes, evt.EventKind(),
		actor.ID, actor.Kind, sessionBytes, evt.CapabilityProof(), string(body),
	)
	if err != nil {
		return fmt.Errorf("sqlite: insert event: %w", err)
	}
	return nil
}

// ReplayRoom returns the most recent `limit` events for the room,
// ascending by HLC. limit <= 0 returns every event.
func (s *Store) ReplayRoom(ctx context.Context, roomID string, limit int) ([]events.Event, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if limit > 0 {
		// Newest N, then reverse to ascending.
		rows, err = s.db.QueryContext(ctx, `
			SELECT envelope_json
			FROM events
			WHERE room_id = ?
			ORDER BY hlc_wall DESC, hlc_logical DESC
			LIMIT ?
		`, roomID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT envelope_json
			FROM events
			WHERE room_id = ?
			ORDER BY hlc_wall ASC, hlc_logical ASC
		`, roomID)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: replay query: %w", err)
	}
	defer rows.Close()

	var out []events.Event
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("sqlite: scan envelope: %w", err)
		}
		evt, err := events.Unmarshal([]byte(body))
		if err != nil {
			return nil, fmt.Errorf("sqlite: unmarshal envelope: %w", err)
		}
		out = append(out, evt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// LIMIT query returned newest-first; reverse to ascending.
	if limit > 0 {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out, nil
}

// LatestHLC returns the highest HLC stamp persisted for the room.
func (s *Store) LatestHLC(ctx context.Context, roomID string) (events.Timestamp, error) {
	var wall, logical sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(hlc_wall), MAX(hlc_logical)
		FROM events
		WHERE room_id = ?
	`, roomID).Scan(&wall, &logical)
	if err != nil {
		return events.Timestamp{}, fmt.Errorf("sqlite: latest hlc: %w", err)
	}
	if !wall.Valid {
		return events.Timestamp{}, nil
	}
	return events.Timestamp{HLC: combineHLC(wall.Int64, logical.Int64)}, nil
}

// Close shuts the underlying SQLite handle.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// splitHLC pulls the upper 48 bits (wall) and lower 16 bits (logical)
// out of a glycerine/hlc.HLC.
func splitHLC(t hlc.HLC) (wall, logical int64) {
	i := int64(t)
	return i & ^int64(0xffff), i & 0xffff
}

// combineHLC reverses splitHLC.
func combineHLC(wall, logical int64) hlc.HLC {
	return hlc.HLC(wall | logical&0xffff)
}

// Assert *Store satisfies store.Store at compile time.
var _ store.Store = (*Store)(nil)

// silence "imported and not used" if all Helper functions disappear.
var _ = binary.BigEndian
