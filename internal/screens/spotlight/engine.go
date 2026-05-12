// Package spotlight's engine is the moment-orchestrator. It cycles items via
// a state machine: presenting → transition → opt-in → presenting. The 5-min
// wall-clock rotation Bogdan shipped in v0.2.0 is replaced by this engine so
// the spotlight feels like a live party rather than a slideshow.
package spotlight

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// engineState is the moment-to-moment status the engine reports.
type engineState int

const (
	// statePresenting means the current idx is on stage. Highlights, chat,
	// repo link, the whole card.
	statePresenting engineState = iota
	// stateTransitioning is the brief gap between two moments. The card is
	// hidden so the field gets the floor.
	stateTransitioning
	// stateOptIn is the 15s window where the next item is previewed and the
	// audience can accept it or skip ahead.
	stateOptIn
)

const (
	presentDuration    = 60 * time.Second
	optInDuration      = 15 * time.Second
	transitionDuration = 1200 * time.Millisecond
)

// Engine cycles items and reports which one is in focus and what kind of
// moment it is. Callers feed it ticks; everything else falls out of state.
type Engine struct {
	state    engineState
	idx      int
	count    int
	deadline time.Time
	now      func() time.Time
}

// NewEngine starts presenting the first item immediately.
func NewEngine(itemsCount int) *Engine {
	e := &Engine{count: itemsCount, now: time.Now}
	e.startPresenting()
	return e
}

// EngineTickMsg fires at 4Hz; the engine only needs sub-second precision since
// its shortest interval is the 1.2s transition.
type EngineTickMsg time.Time

// EngineTickCmd schedules the next engine tick.
func EngineTickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return EngineTickMsg(t)
	})
}

func (e *Engine) startPresenting() {
	e.state = statePresenting
	e.deadline = e.now().Add(presentDuration)
}

// Tick advances the state machine; the second return value is true exactly
// once each time the engine enters statePresenting from another state — the
// caller can use it to fire the entry pulse on the field.
func (e *Engine) Tick(t time.Time) bool {
	if t.Before(e.deadline) {
		return false
	}
	switch e.state {
	case statePresenting:
		if e.count > 0 {
			e.idx = (e.idx + 1) % e.count
		}
		e.state = stateTransitioning
		e.deadline = t.Add(transitionDuration)
	case stateTransitioning:
		e.state = stateOptIn
		e.deadline = t.Add(optInDuration)
	case stateOptIn:
		e.startPresenting()
		return true
	}
	return false
}

// Accept jumps from opt-in (or the brief transition) straight into the next
// presenting moment. Returns true when the jump happens, so the host can pulse.
func (e *Engine) Accept() bool {
	if e.state == stateOptIn || e.state == stateTransitioning {
		e.startPresenting()
		return true
	}
	return false
}

// Skip leaves the current item (whether presenting or about to present) and
// advances to the next one through the normal transition path.
func (e *Engine) Skip() {
	if e.count > 0 {
		e.idx = (e.idx + 1) % e.count
	}
	e.state = stateTransitioning
	e.deadline = e.now().Add(transitionDuration)
}

func (e *Engine) State() engineState       { return e.state }
func (e *Engine) Index() int               { return e.idx }
func (e *Engine) Remaining() time.Duration { return e.deadline.Sub(e.now()) }
func (e *Engine) IsPresenting() bool       { return e.state == statePresenting }
func (e *Engine) IsOptIn() bool            { return e.state == stateOptIn }
func (e *Engine) IsTransition() bool       { return e.state == stateTransitioning }

// OptInProgress returns 0..1 across the 15s opt-in window. 0 means just
// started, 1 means timed out.
func (e *Engine) OptInProgress() float64 {
	if e.state != stateOptIn {
		return 0
	}
	rem := e.deadline.Sub(e.now())
	if rem <= 0 {
		return 1
	}
	return 1 - float64(rem)/float64(optInDuration)
}
