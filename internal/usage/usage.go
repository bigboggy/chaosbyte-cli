// Package usage scans on-disk AI-CLI session data and rolls per-day token
// totals into a uniform shape so the `vibespace report` client can upload
// them to the leaderboard server.
//
// Each reader is responsible for finding its CLI's data directory, parsing
// whatever native format it uses, and emitting []Daily — one row per
// (source, UTC date). Readers must never error just because their CLI isn't
// installed; missing data is reported as a nil/empty slice.
package usage

// Source matches the string values stored in the server's token_usage
// table (store.TokenSource). Kept duplicated here so this package has no
// dependency on internal/store.
type Source string

const (
	Claude   Source = "claude"
	OpenCode Source = "opencode"
	Codex    Source = "codex"
)

// Daily is one (source, date) bucket. Counts are absolute totals for that
// day — the upload protocol is idempotent: re-sending today's count
// overwrites the prior write rather than double-counting.
type Daily struct {
	Source     Source `json:"source"`
	Date       string `json:"date"` // YYYY-MM-DD (UTC)
	Input      int64  `json:"input"`
	Output     int64  `json:"output"`
	CacheWrite int64  `json:"cache_write"`
	CacheRead  int64  `json:"cache_read"`
}

// Total is the all-up token count used for ranking on the leaderboard.
func (d Daily) Total() int64 {
	return d.Input + d.Output + d.CacheWrite + d.CacheRead
}

// Reader is implemented by every AI-CLI scanner.
type Reader interface {
	Source() Source
	// Read returns one Daily per UTC date that has any usage. An empty
	// result with nil error means the CLI is not installed locally — the
	// caller treats that the same as "nothing to upload".
	Read() ([]Daily, error)
}

// AllReaders returns the default reader set with stock paths. Tests and
// custom installs can construct readers directly with explicit roots.
func AllReaders() []Reader {
	return []Reader{
		ClaudeReader{},
		OpenCodeReader{},
		CodexReader{},
	}
}
