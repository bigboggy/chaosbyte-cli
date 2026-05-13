// Package games holds the chat-as-game runtime. A round is a thirty-second
// window in which discrete cascade events fire on specific chat layouts:
// the mod cascades a target word in, players race to type it, each
// matching player gets a cascading +N confirmation, and the winner's
// nick cascade-settles at the end. The cascade is the engine; the game
// is what it scores. There is no background treatment applied to the
// whole chat. Every visible game beat is a foreground cascade on a
// specific Layout via the existing typo macros (Settle / CascadeTo).
package games

import (
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/typo"
)

// DefaultDuration is the length of one round. Thirty seconds gives the
// room time to read the mod's target cascade, react, and post a few
// matches before the winner is named.
const DefaultDuration = 30 * time.Second

// onsetDuration is the ramp-in window at the start of a round. Kept
// around as a beat for the lobby to use when scheduling top-bar fades
// and similar surface ripples; Blitz itself no longer paints anything
// continuous during this window.
const onsetDuration = 1500 * time.Millisecond

// offsetDuration is the ramp-out window at the end of a round. Same
// note as onsetDuration: the winner is resolved at the top of this
// window so the mod's winner cascade has the offset to play through.
const offsetDuration = 1500 * time.Millisecond

// targetWords is the curated word bank a round picks its target from.
// Short, vibe-aligned, single-token. Picked deliberately to land easily
// in normal chat (no contrived "type fizzbuzz" friction).
var targetWords = []string{
	"rust", "zig", "claude", "cursor", "ship", "merge", "fork", "deploy",
	"agent", "vibe", "repo", "smooth", "rough", "tight", "loose", "build",
	"small", "fast", "good", "bad", "hot", "live", "dead", "fix", "land",
}

// Blitz is the active state of one round. The lobby holds a *Blitz while
// a round is in flight, ticks it forward, and forwards new chat events
// for scoring. A round runs three phases:
//
//   - Onset (0 .. onsetDuration): top-of-the-round beat. The lobby uses
//     this window to fade the top bar into game mode and announce the
//     target via a cascading mod ChatAction.
//   - Main (onsetDuration .. duration): the room races to type the
//     target. Each first-time match scores a player; the lobby posts a
//     mod ChatAction confirming the +N.
//   - Offset (duration .. duration+offsetDuration): the winner is
//     resolved at the top of this window so the winner's mod ChatAction
//     cascade-settles inside the offset's tail.
type Blitz struct {
	mu sync.Mutex

	startedAt time.Time
	duration  time.Duration

	target        string
	matchOrder    []string
	scoreByAuthor map[string]int
	lastAuthor    string

	winner    string
	announced bool
	done      bool
}

// NewBlitz starts a round at start and picks a target word at random.
// The lobby reads Target() immediately after to fire the cascading mod
// announcement.
func NewBlitz(start time.Time) *Blitz {
	rng := rand.New(rand.NewSource(start.UnixNano()))
	return &Blitz{
		startedAt:     start,
		duration:      DefaultDuration,
		target:        targetWords[rng.Intn(len(targetWords))],
		matchOrder:    []string{},
		scoreByAuthor: map[string]int{},
	}
}

// Target returns the round's target word. Stable for the lifetime of
// the round.
func (b *Blitz) Target() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.target
}

// Remaining returns the time left in the main phase of the round (before
// the offset window opens). Once main is over the value clamps to zero
// so the top-bar countdown stops at 0:00 rather than going negative.
func (b *Blitz) Remaining(now time.Time) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	elapsed := now.Sub(b.startedAt)
	if elapsed >= b.duration {
		return 0
	}
	return b.duration - elapsed
}

// Tick advances the round's phase. The winner is resolved at the entry
// to the offset window so the lobby can post the winner mod ChatAction
// while the offset window plays through. Done flips only after the
// offset window has fully elapsed so the lobby can keep the *Blitz
// reference alive for the duration of the winner cascade.
func (b *Blitz) Tick(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.done {
		return
	}
	elapsed := now.Sub(b.startedAt)
	if elapsed >= b.duration && b.winner == "" {
		b.winner = b.resolveWinnerLocked()
	}
	if elapsed >= b.duration+offsetDuration {
		b.done = true
	}
}

// Done reports whether the round has fully resolved (including the
// offset window). The lobby clears its *Blitz reference at this point.
func (b *Blitz) Done() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.done
}

// Winner returns the resolved winner. Set at the entry to the offset
// window; empty before that.
func (b *Blitz) Winner() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.winner
}

// WinnerReady returns (winner, true) the first time the winner has been
// resolved but not yet announced. Subsequent calls return ("", false)
// until the next round. The lobby uses this to fire the winner mod
// ChatAction exactly once at the moment the dance starts to dim.
func (b *Blitz) WinnerReady() (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.winner != "" && !b.announced {
		b.announced = true
		return b.winner, true
	}
	return "", false
}

// MatchScore checks whether body contains the round's target word as a
// standalone token and awards points to first-time matchers. Returns
// (points, true) if a new match was recorded, (0, false) otherwise.
// Scoring is 3 / 2 / 1 for the first three unique authors; later
// matches still record but score 1 so persistence still counts a
// little.
func (b *Blitz) MatchScore(author, body string) (int, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.done || b.target == "" || author == "" {
		return 0, false
	}
	if _, already := b.scoreByAuthor[author]; already {
		return 0, false
	}
	if !containsToken(body, b.target) {
		return 0, false
	}
	points := 1
	switch len(b.matchOrder) {
	case 0:
		points = 3
	case 1:
		points = 2
	case 2:
		points = 1
	}
	b.matchOrder = append(b.matchOrder, author)
	b.scoreByAuthor[author] = points
	b.lastAuthor = author
	return points, true
}

// Paint mutates a chat layout's AnimationState to put it into the dance
// state for the round. Every visible chat line gets a moving cascade
// settle frontier so the letters flap-settle through long-ramp glyphs
// and lock into their target chars in a wave that sweeps across each
// line. The wave's phase is per-line so neighbouring rows ripple out of
// sync; the eye reads the whole chat as alive.
//
// On top of the wave, the line is tinted by depth tier: back rows render
// muted, mid rows shimmer between parchment and gold, front rows shimmer
// between phosphor green and gold. The brightness gradient gives the
// surface a sense of depth without leaving the chat substrate.
//
// The mutation rides an intensity envelope (onset → main → offset) so
// the dance ramps in and out at the round boundaries rather than
// snapping on.
func (b *Blitz) Paint(state *typo.AnimationState, layoutIdx, total int, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.done {
		return
	}
	elapsed := now.Sub(b.startedAt)
	intensity := b.intensityLocked(elapsed)
	if intensity <= 0 {
		return
	}
	elapsedSeconds := elapsed.Seconds()

	state.Reveal = 1.0
	state.RevealFromEnd = false

	// CascadeActive renders cells past the settle frontier as random
	// long-ramp glyphs; cells before lock to their target. We oscillate
	// the frontier so each line's wave moves across it back and forth,
	// with per-line phase so neighbouring rows don't ripple in sync.
	state.CascadeActive = true
	cascadePhase := math.Sin(elapsedSeconds*0.9 + float64(layoutIdx)*0.5)
	settle := (cascadePhase + 1.0) / 2.0
	// During onset/offset, fade between full-locked (settle=1) and the
	// oscillating wave so the dance ramps in.
	settle = 1.0 - intensity*(1.0-settle)
	state.CascadeSettle = settle

	depth := 0.5
	if total > 1 {
		depth = float64(layoutIdx) / float64(total-1)
	}
	tintPhase := math.Sin(elapsedSeconds*1.2 + float64(layoutIdx)*0.6)
	state.TintActive = true
	state.TintBlend = intensity
	switch {
	case depth < 0.34:
		state.Tint = theme.Muted
	case depth < 0.67:
		if tintPhase >= 0 {
			state.Tint = theme.Fg
		} else {
			state.Tint = theme.Accent2
		}
	default:
		if tintPhase >= 0 {
			state.Tint = theme.Accent
		} else {
			state.Tint = theme.Accent2
		}
	}
}

// intensityLocked returns the round's intensity in [0, 1]: 0 → 1 over
// onsetDuration, holds at 1 through main, 1 → 0 over offsetDuration,
// then 0 once the round is fully spent. Caller holds the lock.
func (b *Blitz) intensityLocked(elapsed time.Duration) float64 {
	if elapsed < 0 {
		return 0
	}
	if elapsed < onsetDuration {
		return float64(elapsed) / float64(onsetDuration)
	}
	if elapsed < b.duration {
		return 1.0
	}
	if elapsed < b.duration+offsetDuration {
		return 1.0 - float64(elapsed-b.duration)/float64(offsetDuration)
	}
	return 0
}

// Standings returns the current score table as (author, score) pairs
// sorted high to low. The lobby renders this in the top bar during a
// round.
func (b *Blitz) Standings() []Score {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Score, 0, len(b.scoreByAuthor))
	for author, score := range b.scoreByAuthor {
		out = append(out, Score{Author: author, Points: score})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].Points < out[j].Points; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// Score is one row of the live standings board.
type Score struct {
	Author string
	Points int
}

// resolveWinnerLocked picks the round's winner from the score table.
// Ties go to the last scorer; an empty round resolves to "the room" so
// the closing cascade always has a target.
func (b *Blitz) resolveWinnerLocked() string {
	best := 0
	winner := ""
	for author, score := range b.scoreByAuthor {
		if score > best {
			best = score
			winner = author
		}
	}
	if winner == "" {
		winner = b.lastAuthor
	}
	if winner == "" {
		winner = "the room"
	}
	return winner
}

// containsToken returns true if target appears as a whole-token in body
// (case-insensitive, split on non-letter/digit). "i'll ship it" matches
// target "ship"; "shipping" does not.
func containsToken(body, target string) bool {
	if target == "" {
		return false
	}
	lower := strings.ToLower(body)
	want := strings.ToLower(target)
	tokens := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, t := range tokens {
		if t == want {
			return true
		}
	}
	return false
}
