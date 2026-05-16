// Package store is the SQLite-backed persistence layer for everything beyond
// the identity map: cached GitHub profile data, posts, friendships, guestbook.
//
// The identity store (internal/identity) remains the source of truth for the
// SSH-fingerprint -> gh_login mapping. This store keys everything off
// gh_login: it only matters once a session has authenticated.
//
// One *Store is shared across all sessions. Methods are safe for concurrent
// use (SQLite handles locking).
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

// sortLeaderboard ranks entries by total desc; ties break by login asc for a
// deterministic order regardless of map iteration.
func sortLeaderboard(entries []LeaderboardEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Total != entries[j].Total {
			return entries[i].Total > entries[j].Total
		}
		return entries[i].Login < entries[j].Login
	})
}

// Store wraps a *sql.DB with typed helpers for each table. The hot tier
// (in-memory map for today's token_usage rows) batches the per-minute
// upload burst into a single SQLite write at the UTC day rollover; see
// hottier.go for the routing logic.
type Store struct {
	db  *sql.DB
	hot *hotStore
}

// Open opens (or creates) the SQLite database at path and runs migrations.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store: empty path")
	}
	// Ensure parent dir exists when path is nested.
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		// best-effort; sqlite will fail loudly if it can't open.
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite is single-writer; cap connections to avoid spurious busy errors.
	db.SetMaxOpenConns(1)
	s := &Store{db: db, hot: newHotStore()}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			gh_login     TEXT PRIMARY KEY,
			name         TEXT NOT NULL DEFAULT '',
			bio          TEXT NOT NULL DEFAULT '',
			avatar_url   TEXT NOT NULL DEFAULT '',
			location     TEXT NOT NULL DEFAULT '',
			company      TEXT NOT NULL DEFAULT '',
			followers    INTEGER NOT NULL DEFAULT 0,
			following    INTEGER NOT NULL DEFAULT 0,
			public_repos INTEGER NOT NULL DEFAULT 0,
			synced_at    INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS repos (
			gh_login    TEXT NOT NULL,
			name        TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			language    TEXT NOT NULL DEFAULT '',
			stars       INTEGER NOT NULL DEFAULT 0,
			forks       INTEGER NOT NULL DEFAULT 0,
			is_fork     INTEGER NOT NULL DEFAULT 0,
			updated_at  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (gh_login, name)
		)`,
		`CREATE INDEX IF NOT EXISTS repos_by_login ON repos(gh_login, stars DESC, updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS stars (
			gh_login    TEXT NOT NULL,
			full_name   TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			language    TEXT NOT NULL DEFAULT '',
			stars       INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (gh_login, full_name)
		)`,
		`CREATE TABLE IF NOT EXISTS contributions (
			gh_login TEXT NOT NULL,
			date     TEXT NOT NULL,
			count    INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (gh_login, date)
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			author     TEXT NOT NULL,
			body       TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS posts_by_author ON posts(author, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS friendships (
			requester   TEXT NOT NULL,
			target      TEXT NOT NULL,
			status      TEXT NOT NULL,
			created_at  INTEGER NOT NULL,
			accepted_at INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (requester, target)
		)`,
		`CREATE INDEX IF NOT EXISTS friendships_by_target ON friendships(target, status)`,
		`CREATE TABLE IF NOT EXISTS guestbook (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_owner TEXT NOT NULL,
			author        TEXT NOT NULL,
			body          TEXT NOT NULL,
			created_at    INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS guestbook_by_owner ON guestbook(profile_owner, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS token_usage (
			gh_login    TEXT NOT NULL,
			source      TEXT NOT NULL,
			date        TEXT NOT NULL,
			input_toks  INTEGER NOT NULL DEFAULT 0,
			output_toks INTEGER NOT NULL DEFAULT 0,
			cache_write INTEGER NOT NULL DEFAULT 0,
			cache_read  INTEGER NOT NULL DEFAULT 0,
			updated_at  INTEGER NOT NULL,
			PRIMARY KEY (gh_login, source, date)
		)`,
		`CREATE INDEX IF NOT EXISTS token_usage_by_date ON token_usage(date)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w: %s", err, stmt)
		}
	}
	return nil
}

// ── Users ───────────────────────────────────────────────────────────────────

// User is the cached GitHub profile.
type User struct {
	Login       string
	Name        string
	Bio         string
	AvatarURL   string
	Location    string
	Company     string
	Followers   int
	Following   int
	PublicRepos int
	SyncedAt    time.Time
}

// UpsertUser inserts or replaces a cached profile row.
func (s *Store) UpsertUser(u User) error {
	_, err := s.db.Exec(`
		INSERT INTO users (gh_login, name, bio, avatar_url, location, company,
		                   followers, following, public_repos, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(gh_login) DO UPDATE SET
			name         = excluded.name,
			bio          = excluded.bio,
			avatar_url   = excluded.avatar_url,
			location     = excluded.location,
			company      = excluded.company,
			followers    = excluded.followers,
			following    = excluded.following,
			public_repos = excluded.public_repos,
			synced_at    = excluded.synced_at`,
		u.Login, u.Name, u.Bio, u.AvatarURL, u.Location, u.Company,
		u.Followers, u.Following, u.PublicRepos, u.SyncedAt.Unix(),
	)
	return err
}

// User returns the cached profile for login. Returns (zero, false) if missing.
func (s *Store) User(login string) (User, bool, error) {
	var u User
	var synced int64
	err := s.db.QueryRow(`
		SELECT gh_login, name, bio, avatar_url, location, company,
		       followers, following, public_repos, synced_at
		FROM users WHERE gh_login = ?`, login,
	).Scan(&u.Login, &u.Name, &u.Bio, &u.AvatarURL, &u.Location, &u.Company,
		&u.Followers, &u.Following, &u.PublicRepos, &synced)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	u.SyncedAt = time.Unix(synced, 0)
	return u, true, nil
}

// ── Repos ───────────────────────────────────────────────────────────────────

// Repo is one of a user's public repos.
type Repo struct {
	Name        string
	Description string
	Language    string
	Stars       int
	Forks       int
	IsFork      bool
	UpdatedAt   time.Time
}

// ReplaceRepos atomically replaces the cached repo list for login.
func (s *Store) ReplaceRepos(login string, repos []Repo) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM repos WHERE gh_login = ?`, login); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO repos
		(gh_login, name, description, language, stars, forks, is_fork, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range repos {
		fork := 0
		if r.IsFork {
			fork = 1
		}
		if _, err := stmt.Exec(login, r.Name, r.Description, r.Language,
			r.Stars, r.Forks, fork, r.UpdatedAt.Unix()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// TopRepos returns up to limit of login's repos, ordered by stars then recency.
// Forks are excluded — we want originals on the profile.
func (s *Store) TopRepos(login string, limit int) ([]Repo, error) {
	rows, err := s.db.Query(`
		SELECT name, description, language, stars, forks, is_fork, updated_at
		FROM repos
		WHERE gh_login = ? AND is_fork = 0
		ORDER BY stars DESC, updated_at DESC
		LIMIT ?`, login, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Repo
	for rows.Next() {
		var r Repo
		var fork int
		var updated int64
		if err := rows.Scan(&r.Name, &r.Description, &r.Language,
			&r.Stars, &r.Forks, &fork, &updated); err != nil {
			return nil, err
		}
		r.IsFork = fork != 0
		r.UpdatedAt = time.Unix(updated, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ── Stars ───────────────────────────────────────────────────────────────────

// StarredRepo is a repo (owned by anyone) that login starred.
type StarredRepo struct {
	FullName    string // owner/repo
	Description string
	Language    string
	Stars       int
}

// ReplaceStars atomically replaces the starred-repo list for login.
func (s *Store) ReplaceStars(login string, stars []StarredRepo) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM stars WHERE gh_login = ?`, login); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO stars
		(gh_login, full_name, description, language, stars)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range stars {
		if _, err := stmt.Exec(login, r.FullName, r.Description, r.Language, r.Stars); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// TopStars returns up to limit of login's most-starred starred repos.
func (s *Store) TopStars(login string, limit int) ([]StarredRepo, error) {
	rows, err := s.db.Query(`
		SELECT full_name, description, language, stars
		FROM stars
		WHERE gh_login = ?
		ORDER BY stars DESC
		LIMIT ?`, login, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StarredRepo
	for rows.Next() {
		var r StarredRepo
		if err := rows.Scan(&r.FullName, &r.Description, &r.Language, &r.Stars); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// StarsCount returns the number of starred repos cached for login.
func (s *Store) StarsCount(login string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM stars WHERE gh_login = ?`, login).Scan(&n)
	return n, err
}

// ── Contributions ───────────────────────────────────────────────────────────

// ContribDay is one day of activity counts.
type ContribDay struct {
	Date  time.Time
	Count int
}

// ReplaceContributions atomically replaces the contribution history for login.
// Days with zero counts may be omitted; profile rendering tolerates gaps.
func (s *Store) ReplaceContributions(login string, days []ContribDay) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM contributions WHERE gh_login = ?`, login); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO contributions (gh_login, date, count) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, d := range days {
		if _, err := stmt.Exec(login, d.Date.Format("2006-01-02"), d.Count); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Contributions returns days for login in [from, to] inclusive, sorted ascending.
func (s *Store) Contributions(login string, from, to time.Time) ([]ContribDay, error) {
	rows, err := s.db.Query(`
		SELECT date, count FROM contributions
		WHERE gh_login = ? AND date >= ? AND date <= ?
		ORDER BY date ASC`,
		login, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContribDay
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			continue
		}
		out = append(out, ContribDay{Date: t, Count: n})
	}
	return out, rows.Err()
}

// ── Posts ───────────────────────────────────────────────────────────────────

// Post is one entry on a user's profile feed.
type Post struct {
	ID        int64
	Author    string
	Body      string
	CreatedAt time.Time
}

// CreatePost inserts a post and returns its id.
func (s *Store) CreatePost(author, body string) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO posts (author, body, created_at) VALUES (?, ?, ?)`,
		author, body, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// PostsByAuthor returns the most-recent up-to-limit posts by author.
func (s *Store) PostsByAuthor(author string, limit int) ([]Post, error) {
	rows, err := s.db.Query(`
		SELECT id, author, body, created_at FROM posts
		WHERE author = ?
		ORDER BY created_at DESC
		LIMIT ?`, author, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Post
	for rows.Next() {
		var p Post
		var t int64
		if err := rows.Scan(&p.ID, &p.Author, &p.Body, &t); err != nil {
			return nil, err
		}
		p.CreatedAt = time.Unix(t, 0)
		out = append(out, p)
	}
	return out, rows.Err()
}

// ── Friendships ─────────────────────────────────────────────────────────────

// FriendStatus describes the relationship between a viewer and a target from
// the viewer's perspective.
type FriendStatus int

const (
	FriendNone       FriendStatus = iota // no row exists
	FriendPendingOut                     // viewer sent a request; target hasn't accepted
	FriendPendingIn                      // target sent a request; viewer can accept
	FriendAccepted                       // mutual friends
	FriendSelf                           // viewer == target
)

// FriendStatusBetween reports the friend state between viewer and target.
func (s *Store) FriendStatusBetween(viewer, target string) (FriendStatus, error) {
	if viewer == "" || target == "" {
		return FriendNone, nil
	}
	if viewer == target {
		return FriendSelf, nil
	}
	// Look for either direction.
	var requester, status string
	err := s.db.QueryRow(`
		SELECT requester, status FROM friendships
		WHERE (requester = ? AND target = ?) OR (requester = ? AND target = ?)
		LIMIT 1`,
		viewer, target, target, viewer,
	).Scan(&requester, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return FriendNone, nil
	}
	if err != nil {
		return FriendNone, err
	}
	if status == "accepted" {
		return FriendAccepted, nil
	}
	// pending
	if requester == viewer {
		return FriendPendingOut, nil
	}
	return FriendPendingIn, nil
}

// AreFriends is a convenience for the guestbook gate.
func (s *Store) AreFriends(a, b string) (bool, error) {
	if a == "" || b == "" || a == b {
		return false, nil
	}
	st, err := s.FriendStatusBetween(a, b)
	return st == FriendAccepted, err
}

// SendFriendRequest creates a pending row from requester -> target. Returns an
// error if a row already exists in either direction.
func (s *Store) SendFriendRequest(requester, target string) error {
	if requester == "" || target == "" {
		return errors.New("empty login")
	}
	if requester == target {
		return errors.New("cannot friend yourself")
	}
	st, err := s.FriendStatusBetween(requester, target)
	if err != nil {
		return err
	}
	switch st {
	case FriendAccepted:
		return errors.New("already friends")
	case FriendPendingOut:
		return errors.New("request already pending")
	case FriendPendingIn:
		return errors.New("they already sent you a request — accept it instead")
	}
	_, err = s.db.Exec(`INSERT INTO friendships
		(requester, target, status, created_at) VALUES (?, ?, 'pending', ?)`,
		requester, target, time.Now().Unix())
	return err
}

// AcceptFriendRequest marks a pending row addressed to viewer (from requester)
// as accepted. Returns an error if no pending row exists.
func (s *Store) AcceptFriendRequest(viewer, requester string) error {
	res, err := s.db.Exec(`UPDATE friendships
		SET status = 'accepted', accepted_at = ?
		WHERE requester = ? AND target = ? AND status = 'pending'`,
		time.Now().Unix(), requester, viewer)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("no pending request from that user")
	}
	return nil
}

// RemoveFriendship removes any row in either direction between a and b.
// Used for both rejecting a pending request and unfriending.
func (s *Store) RemoveFriendship(a, b string) error {
	_, err := s.db.Exec(`DELETE FROM friendships
		WHERE (requester = ? AND target = ?) OR (requester = ? AND target = ?)`,
		a, b, b, a)
	return err
}

// Friend is one row in a user's friend list.
type Friend struct {
	Login string
	Since time.Time
}

// Friends returns the accepted friends of login.
func (s *Store) Friends(login string) ([]Friend, error) {
	rows, err := s.db.Query(`
		SELECT CASE WHEN requester = ? THEN target ELSE requester END AS friend,
		       accepted_at
		FROM friendships
		WHERE status = 'accepted' AND (requester = ? OR target = ?)
		ORDER BY accepted_at DESC`, login, login, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Friend
	for rows.Next() {
		var f Friend
		var t int64
		if err := rows.Scan(&f.Login, &t); err != nil {
			return nil, err
		}
		f.Since = time.Unix(t, 0)
		out = append(out, f)
	}
	return out, rows.Err()
}

// IncomingRequests returns pending requests addressed to login.
func (s *Store) IncomingRequests(login string) ([]Friend, error) {
	rows, err := s.db.Query(`
		SELECT requester, created_at FROM friendships
		WHERE target = ? AND status = 'pending'
		ORDER BY created_at DESC`, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Friend
	for rows.Next() {
		var f Friend
		var t int64
		if err := rows.Scan(&f.Login, &t); err != nil {
			return nil, err
		}
		f.Since = time.Unix(t, 0)
		out = append(out, f)
	}
	return out, rows.Err()
}

// OutgoingRequests returns pending requests sent by login.
func (s *Store) OutgoingRequests(login string) ([]Friend, error) {
	rows, err := s.db.Query(`
		SELECT target, created_at FROM friendships
		WHERE requester = ? AND status = 'pending'
		ORDER BY created_at DESC`, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Friend
	for rows.Next() {
		var f Friend
		var t int64
		if err := rows.Scan(&f.Login, &t); err != nil {
			return nil, err
		}
		f.Since = time.Unix(t, 0)
		out = append(out, f)
	}
	return out, rows.Err()
}

// ── Guestbook ───────────────────────────────────────────────────────────────

// GuestbookEntry is one signed message on someone's profile.
type GuestbookEntry struct {
	ID        int64
	Owner     string
	Author    string
	Body      string
	CreatedAt time.Time
}

// SignGuestbook inserts an entry. The caller is responsible for enforcing the
// "must be friends" gate (so we can produce a friendly error message before
// hitting SQL).
func (s *Store) SignGuestbook(owner, author, body string) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO guestbook (profile_owner, author, body, created_at)
		VALUES (?, ?, ?, ?)`, owner, author, body, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ── Token usage (leaderboard) ───────────────────────────────────────────────

// TokenSource is a stable id for the AI CLI the tokens were spent in. Stored
// as the literal lowercase string in the `source` column so the value is
// stable across migrations.
type TokenSource string

const (
	SourceClaude   TokenSource = "claude"
	SourceOpenCode TokenSource = "opencode"
	SourceCodex    TokenSource = "codex"
)

// TokenUsage is one (user, source, day) bucket. Days are stored as
// YYYY-MM-DD UTC text so a string PK is stable and easy to range-query.
type TokenUsage struct {
	Login       string
	Source      TokenSource
	Date        string // YYYY-MM-DD (UTC)
	Input       int64
	Output      int64
	CacheWrite  int64
	CacheRead   int64
	UpdatedAt   time.Time
}

// Total returns the all-up token count summed across input/output/cache —
// used as the single ranking number on the leaderboard.
func (u TokenUsage) Total() int64 {
	return u.Input + u.Output + u.CacheWrite + u.CacheRead
}

// RecordTokenUsage upserts one (login, source, date) bucket with absolute
// totals. Today's bucket goes to the in-memory hot tier; older dates write
// straight to SQLite. The hot tier coalesces the per-minute upload burst
// into a single SQLite write at the UTC day rollover. See hottier.go.
//
// Callers should send the daily total, not deltas — re-uploading the same
// (user, source, day) overwrites the prior value rather than accumulating.
func (s *Store) RecordTokenUsage(u TokenUsage) error {
	if u.Login == "" || u.Source == "" || u.Date == "" {
		return errors.New("store: token usage missing login/source/date")
	}
	todayWritten, stale, oldDay := s.hot.upsert(u)
	// Flush a rolled-over prior day to SQLite. Caller is unaware of the
	// rollover — they just see RecordTokenUsage succeed.
	for _, e := range stale {
		if err := s.writeColdTokenUsage(e); err != nil {
			return err
		}
	}
	// Late/backfilled data for a past day persists directly.
	for _, e := range oldDay {
		if err := s.writeColdTokenUsage(e); err != nil {
			return err
		}
	}
	_ = todayWritten
	return nil
}

// writeColdTokenUsage is the raw SQLite upsert. The hot tier calls it
// during rollover; backfill writes (date < today) call it directly. Stays
// idempotent on (login, source, date) so re-runs from the hot tier are
// safe if FlushHot is invoked twice.
func (s *Store) writeColdTokenUsage(u TokenUsage) error {
	_, err := s.db.Exec(`
		INSERT INTO token_usage (gh_login, source, date, input_toks, output_toks, cache_write, cache_read, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(gh_login, source, date) DO UPDATE SET
			input_toks  = excluded.input_toks,
			output_toks = excluded.output_toks,
			cache_write = excluded.cache_write,
			cache_read  = excluded.cache_read,
			updated_at  = excluded.updated_at`,
		u.Login, string(u.Source), u.Date,
		u.Input, u.Output, u.CacheWrite, u.CacheRead,
		time.Now().Unix())
	return err
}

// FlushHot drains the hot tier into SQLite. Called by the server's SIGTERM
// shutdown path so a graceful stop persists today's accumulated totals
// before the process exits. Idempotent — running it twice in a row is a
// no-op after the first call drains the map.
func (s *Store) FlushHot() error {
	if s == nil || s.hot == nil {
		return nil
	}
	for _, e := range s.hot.drain() {
		if err := s.writeColdTokenUsage(e); err != nil {
			return err
		}
	}
	return nil
}

// LeaderboardEntry is one user's aggregated totals for the queried window.
// PerSource is broken out so the full leaderboard can show "claude / opencode
// / codex" splits next to the headline total.
type LeaderboardEntry struct {
	Login     string
	Total     int64
	PerSource map[TokenSource]int64
}

// Leaderboard returns the top-`limit` users by total tokens since `since`
// (inclusive, UTC day boundary). Pass a zero `since` to mean all-time.
//
// Reads merge the cold tier (SQLite, historical) with the hot tier
// (in-memory, today). The hot snapshot is taken once per call so the read
// is point-in-time consistent even while concurrent uploads update the
// map.
func (s *Store) Leaderboard(since time.Time, limit int) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 5
	}
	sinceStr := ""
	if !since.IsZero() {
		sinceStr = since.UTC().Format("2006-01-02")
	}

	agg := map[string]*LeaderboardEntry{}

	// Cold tier: SQLite. One row per (user, source) aggregated by SUM so
	// per-day rows collapse before they hit Go.
	var args []any
	where := ""
	if sinceStr != "" {
		where = "WHERE date >= ?"
		args = append(args, sinceStr)
	}
	q := fmt.Sprintf(`
		SELECT gh_login, source,
		       COALESCE(SUM(input_toks),  0),
		       COALESCE(SUM(output_toks), 0),
		       COALESCE(SUM(cache_write), 0),
		       COALESCE(SUM(cache_read),  0)
		FROM token_usage
		%s
		GROUP BY gh_login, source`, where)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var login, src string
		var in, out, cw, cr int64
		if err := rows.Scan(&login, &src, &in, &out, &cw, &cr); err != nil {
			return nil, err
		}
		addLeaderboardRow(agg, login, TokenSource(src), in+out+cw+cr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Hot tier: today's in-memory buckets. Skip if `since` rules out today,
	// otherwise fold them into the same aggregate as the SQLite rows.
	if sinceStr == "" || todayUTC() >= sinceStr {
		for _, h := range s.hot.snapshot() {
			if sinceStr != "" && h.Date < sinceStr {
				continue
			}
			addLeaderboardRow(agg, h.Login, h.Source, h.Total())
		}
	}

	out := make([]LeaderboardEntry, 0, len(agg))
	for _, e := range agg {
		out = append(out, *e)
	}
	// Stable order: total desc, then login asc for deterministic ties.
	sortLeaderboard(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// addLeaderboardRow merges one (login, source, total) into the aggregate
// map shared by the cold-tier scan and the hot-tier merge. Reused so both
// paths follow the exact same accumulation rules.
func addLeaderboardRow(agg map[string]*LeaderboardEntry, login string, src TokenSource, total int64) {
	e, ok := agg[login]
	if !ok {
		e = &LeaderboardEntry{Login: login, PerSource: map[TokenSource]int64{}}
		agg[login] = e
	}
	e.Total += total
	e.PerSource[src] += total
}

// Guestbook returns the most-recent up-to-limit entries on owner's profile.
func (s *Store) Guestbook(owner string, limit int) ([]GuestbookEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, profile_owner, author, body, created_at
		FROM guestbook
		WHERE profile_owner = ?
		ORDER BY created_at DESC
		LIMIT ?`, owner, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GuestbookEntry
	for rows.Next() {
		var e GuestbookEntry
		var t int64
		if err := rows.Scan(&e.ID, &e.Owner, &e.Author, &e.Body, &t); err != nil {
			return nil, err
		}
		e.CreatedAt = time.Unix(t, 0)
		out = append(out, e)
	}
	return out, rows.Err()
}
