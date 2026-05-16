package store

import (
	"sync"
	"time"
)

// hotStore holds today's per-(login, source) token totals in memory. Every
// minute, dozens of `vibespace report` clients re-push the same (login,
// source, today) bucket — landing those in SQLite over and over would burn
// IO and lock contention. The hot tier collapses them into a single map
// upsert, then flushes once at the UTC day rollover.
//
// Crash recovery: the hot tier is intentionally *not* persisted. Clients
// always push their full transcript history on every report, so within
// ~60s of restart the hot tier self-heals from the next upload. Yesterday
// and earlier are already durable in SQLite. Worst case: a few minutes of
// "today" totals show stale on the leaderboard until the first post-restart
// upload arrives — usually a single client cycle.
type hotStore struct {
	mu      sync.Mutex
	day     string // UTC date (YYYY-MM-DD) the buckets below belong to
	buckets map[hotKey]TokenUsage
}

type hotKey struct {
	Login  string
	Source TokenSource
}

func newHotStore() *hotStore {
	return &hotStore{
		day:     todayUTC(),
		buckets: map[hotKey]TokenUsage{},
	}
}

// upsert records the entry in the hot tier. The return value tells the
// caller what to do with stray data:
//
//   - todayWritten=true means u was for the current hot day and now sits in
//     memory; no SQLite work needed.
//   - stale is non-nil when the upsert detected a day rollover (u.Date >
//     hot.day): it contains the prior day's flushed buckets so the caller
//     can write them to SQLite. The hot tier has already advanced to the
//     new day and stored u.
//   - oldDayDirect is non-nil when u.Date < hot.day (late or backfilled
//     data for an earlier day). It contains exactly {u} and the hot tier
//     is unchanged — caller writes directly to SQLite.
func (h *hotStore) upsert(u TokenUsage) (todayWritten bool, stale []TokenUsage, oldDayDirect []TokenUsage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	switch {
	case u.Date == h.day:
		h.buckets[hotKey{u.Login, u.Source}] = u
		return true, nil, nil
	case u.Date > h.day:
		// Day rolled over. Drain the prior day, advance, then store u.
		stale = h.drainLocked()
		h.day = u.Date
		h.buckets[hotKey{u.Login, u.Source}] = u
		return true, stale, nil
	default:
		// u is for a past day — don't touch hot, let caller persist directly.
		return false, nil, []TokenUsage{u}
	}
}

// snapshot returns a copy of the current buckets. Used by Leaderboard reads
// to merge today's in-memory totals with SQLite's historical data.
func (h *hotStore) snapshot() []TokenUsage {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]TokenUsage, 0, len(h.buckets))
	for _, b := range h.buckets {
		out = append(out, b)
	}
	return out
}

// drain returns the current buckets and clears the map. Used by FlushHot
// on shutdown so today's accumulated totals are persisted before exit.
func (h *hotStore) drain() []TokenUsage {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.drainLocked()
}

func (h *hotStore) drainLocked() []TokenUsage {
	out := make([]TokenUsage, 0, len(h.buckets))
	for _, b := range h.buckets {
		out = append(out, b)
	}
	h.buckets = map[hotKey]TokenUsage{}
	return out
}

// todayUTC returns the current UTC date in the same format used by the
// `date` column of token_usage. Module-level so tests / future migrations
// can swap it without bubbling a clock dependency through the Store API.
func todayUTC() string {
	return time.Now().UTC().Format("2006-01-02")
}
