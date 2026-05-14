-- 0001_init.sql: the event log table that every Phase 1 topic writes
-- into. PRIMARY KEY (room_id, hlc_wall, hlc_logical) guarantees unique
-- ordering per room. WITHOUT ROWID skips SQLite's implicit rowid since
-- the natural key is our access path.
--
-- The hlc_wall + hlc_logical split mirrors the glycerine/hlc int64
-- layout: upper 48 bits as nanoseconds since epoch, lower 16 bits as
-- the logical counter. Storing them as separate INTEGER columns makes
-- range scans cheap.

CREATE TABLE IF NOT EXISTS events (
    room_id        TEXT    NOT NULL,
    hlc_wall       INTEGER NOT NULL,
    hlc_logical    INTEGER NOT NULL,
    event_id       BLOB    NOT NULL,
    kind           TEXT    NOT NULL,
    actor_id       TEXT    NOT NULL,
    actor_kind     TEXT    NOT NULL,
    session_id     BLOB    NOT NULL,
    capability_raw BLOB,
    envelope_json  TEXT    NOT NULL,
    inserted_at    INTEGER NOT NULL DEFAULT (strftime('%s', 'now') * 1000),
    PRIMARY KEY (room_id, hlc_wall, hlc_logical)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_events_by_room_kind ON events (room_id, kind, hlc_wall);
CREATE INDEX IF NOT EXISTS idx_events_by_actor     ON events (actor_id, hlc_wall);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);

INSERT OR IGNORE INTO schema_migrations (version, applied_at)
VALUES (1, strftime('%s', 'now') * 1000);
